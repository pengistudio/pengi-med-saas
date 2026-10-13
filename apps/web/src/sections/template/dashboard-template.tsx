import { SelectLanguage, useText } from "@pengi/shared";
import {
	AppShell,
	Avatar,
	AvatarFallback,
	AvatarImage,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@pengi/ui";
import {
	AlertTriangle,
	Building2,
	CreditCard,
	HelpCircle,
	Power,
} from "lucide-react";
import {
	Suspense,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { Outlet, useLocation, useNavigate } from "react-router";
import { getPendingReviewCount } from "@/api/exam-order-service";
import { initiatePayment } from "@/api/subscription-service";
import NotificationBell from "@/components/custom/notification-bell";
import { PageSkeleton } from "@/components/custom/page-skeleton";
import { DoctorReviewBanner } from "@/components/features/doctors/doctor-notices";
import { DoctorOnboardingDialog } from "@/components/features/doctors/doctor-onboarding-dialog";
import {
	createNavItems,
	type EnabledFeatures,
	filterNavItems,
	type NavBadgeKey,
	pickBottomNav,
} from "@/config/nav-config";
import useAuth from "@/hooks/use-auth";
import { useNotificationsPoll } from "@/hooks/use-notifications-poll";
import usePermission from "@/hooks/use-permission";
import { usePolling } from "@/hooks/use-polling";
import { PERMISSIONS } from "@/lib/constants";
import { getPageTitle } from "@/lib/page-title";
import { refreshEnvironment } from "@/lib/refresh-environment";
import {
	selectEnvironment,
	selectSubscriptionExpired,
	selectSubscriptionGraceDaysLeft,
	useSessionStore,
} from "@/store/session-store";
import {
	refreshWhatsAppUnread,
	selectUnreadConversations,
	useWhatsAppStore,
} from "@/store/whatsapp-store";

/** Faster than the bell's poll: more unread messages also reload the bell (new-message toast). */
const WHATSAPP_UNREAD_POLL_MS = 15_000;

/** The frame of every signed-in page; the route tree renders pages in its Outlet. */
export function DashboardLayout() {
	const { logout } = useAuth();
	const { textGet } = useText();
	const navigate = useNavigate();
	const [paying, setPaying] = useState(false);
	const reloadNotifications = useNotificationsPoll();
	// Persisted permissions can predate a deploy: re-read them once per load.
	useEffect(() => {
		refreshEnvironment();
	}, []);

	const handlePay = useCallback(async () => {
		setPaying(true);
		const res = await initiatePayment();
		setPaying(false);
		if (res.success) {
			window.open(res.data.checkout_url, "_blank");
		}
	}, []);
	const { checkPermission } = usePermission();

	const { pathname } = useLocation();
	const isSubscriptionPage = pathname === "/subscription";

	const environment = useSessionStore(selectEnvironment);
	const subscriptionExpired = useSessionStore(selectSubscriptionExpired);
	const graceDaysLeft = useSessionStore(selectSubscriptionGraceDaysLeft);

	const initials = environment?.name
		? environment.name
				.split(" ")
				.map((n) => n[0])
				.join("")
		: "";

	// Parse enabled features from environment
	const rawEnabledFeatures = environment?.enabled_features;
	const enabledFeatures: EnabledFeatures = useMemo(() => {
		if (!rawEnabledFeatures) {
			return { clinical: true, billing: true, team: true };
		}
		try {
			return JSON.parse(rawEnabledFeatures);
		} catch {
			return { clinical: true, billing: true, team: true };
		}
	}, [rawEnabledFeatures]);

	const unfilteredNavItems = useMemo(() => createNavItems(textGet), [textGet]);
	const allNavItems = useMemo(
		() => filterNavItems(unfilteredNavItems, checkPermission, enabledFeatures),
		[unfilteredNavItems, enabledFeatures, checkPermission],
	);
	const pageTitle =
		getPageTitle(unfilteredNavItems, pathname, textGet) ??
		textGet("dashboard.title");
	const canReviewExams =
		checkPermission([PERMISSIONS.EXAM_ORDERS.PERMISSION_READ_EXAM_ORDER]) &&
		checkPermission([PERMISSIONS.EXAM_ORDERS.PERMISSION_REVIEW_EXAM_RESULTS]);
	const [pendingExamReviews, setPendingExamReviews] = useState(0);
	// Once per route change, no polling; pathname is the trigger, not an input.
	useEffect(() => {
		if (!canReviewExams) {
			setPendingExamReviews(0);
			return;
		}
		let cancelled = false;
		getPendingReviewCount().then((n) => {
			if (!cancelled) setPendingExamReviews(n);
		});
		return () => {
			cancelled = true;
		};
	}, [canReviewExams, pathname]);
	const canUseInbox =
		checkPermission([PERMISSIONS.WHATSAPP.PERMISSION_USE_WHATSAPP_INBOX]) &&
		enabledFeatures.whatsapp !== false;
	const whatsappUnread = useWhatsAppStore(selectUnreadConversations);
	useEffect(() => {
		if (canUseInbox) refreshWhatsAppUnread();
	}, [canUseInbox]);
	usePolling(refreshWhatsAppUnread, WHATSAPP_UNREAD_POLL_MS, canUseInbox);
	// A new WhatsApp message shows up here first: reload the bell now so its
	// toast doesn't wait for the next notifications poll.
	const whatsappUnreadMessages = useWhatsAppStore((s) => s.unreadMessages);
	const previousUnreadMessages = useRef(whatsappUnreadMessages);
	useEffect(() => {
		if (whatsappUnreadMessages > previousUnreadMessages.current) {
			reloadNotifications();
		}
		previousUnreadMessages.current = whatsappUnreadMessages;
	}, [whatsappUnreadMessages, reloadNotifications]);
	const badges: Record<NavBadgeKey, number> = {
		pendingExamReviews,
		whatsappUnread: canUseInbox ? whatsappUnread : 0,
	};
	const withBadges = <T extends { badgeKey?: NavBadgeKey }>(items: T[]) =>
		items.map((item) =>
			item.badgeKey ? { ...item, badge: badges[item.badgeKey] } : item,
		);
	const navItems = withBadges(allNavItems.filter((item) => !item.isBottom));
	const bottomNavItems = withBadges(pickBottomNav(allNavItems, textGet));
	const footerNavItems = [
		...allNavItems.filter((item) => item.isBottom),
		{
			label: textGet("dashboard.help"),
			href: "/",
			icon: HelpCircle,
			matchActive: false,
		},
	];

	const showExpiredWall =
		subscriptionExpired && !isSubscriptionPage && graceDaysLeft === 0;

	return (
		<AppShell
			brand={{ icon: Building2, name: environment?.trade_name }}
			nav={navItems}
			footerNav={footerNavItems}
			matchNested
			bottomNav={bottomNavItems}
			title={pageTitle}
			actions={
				<>
					<SelectLanguage />
					<NotificationBell />
					<DropdownMenu>
						<DropdownMenuTrigger className="relative h-10 w-10 cursor-pointer rounded-full">
							<Avatar className="h-10 w-10">
								<AvatarImage alt="User" />
								<AvatarFallback>{initials}</AvatarFallback>
							</Avatar>
						</DropdownMenuTrigger>
						<DropdownMenuContent align="end" className="w-56">
							<DropdownMenuGroup>
								<DropdownMenuLabel>
									{textGet("dashboard.dropdown.profile.title")}
								</DropdownMenuLabel>
								<DropdownMenuSeparator />
								<DropdownMenuItem onClick={() => navigate("/profile")}>
									{textGet("dashboard.dropdown.profile")}
								</DropdownMenuItem>
								<DropdownMenuItem onClick={() => navigate("/settings")}>
									{textGet("dashboard.dropdown.settings")}
								</DropdownMenuItem>
							</DropdownMenuGroup>
							<DropdownMenuSeparator />
							<DropdownMenuItem onClick={logout} variant="destructive">
								<Power className="w-4 h-4 text-red-500" />
								{textGet("dashboard.dropdown.logout")}
							</DropdownMenuItem>
						</DropdownMenuContent>
					</DropdownMenu>
				</>
			}
			banner={
				<>
					{graceDaysLeft < 0 && !isSubscriptionPage && (
						<div className="flex shrink-0 flex-wrap items-center justify-between gap-x-3 gap-y-2 border-b border-amber-500/30 bg-amber-500/10 px-4 py-2.5">
							<div className="flex items-center gap-2">
								<AlertTriangle className="h-4 w-4 shrink-0 text-amber-600" />
								<p className="text-sm font-medium text-amber-700 dark:text-amber-400">
									{textGet("subscription.grace.banner")}{" "}
									<span className="font-bold">
										{3 + graceDaysLeft}{" "}
										{textGet("subscription.grace.days_remaining")}
									</span>
								</p>
							</div>
							<Button
								size="sm"
								variant="outline"
								className="shrink-0 border-amber-500/50 text-amber-700 hover:bg-amber-500/10 dark:text-amber-400"
								onClick={() => navigate("/subscription")}
							>
								{textGet("subscription.grace.cta")}
							</Button>
						</div>
					)}
					{enabledFeatures.clinical !== false && <DoctorReviewBanner />}
				</>
			}
		>
			{showExpiredWall ? (
				<div className="flex h-full items-center justify-center">
					<Card className="w-full max-w-md border-destructive/50">
						<CardHeader className="text-center">
							<div className="mb-2 flex justify-center">
								<AlertTriangle className="h-12 w-12 text-destructive" />
							</div>
							<CardTitle className="text-destructive">
								{textGet("subscription.expired.title")}
							</CardTitle>
							<CardDescription>
								{textGet("subscription.expired.description")}
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-2 text-center">
							<p className="text-sm text-muted-foreground">
								{textGet("subscription.expired.contact")}
							</p>
							<Button className="w-full" onClick={handlePay} disabled={paying}>
								<CreditCard className="mr-2 h-4 w-4" />
								{textGet("dashboard.subscription.pay_now")}
							</Button>
							<Button variant="outline" className="w-full" onClick={logout}>
								<Power className="mr-2 h-4 w-4" />
								{textGet("dashboard.dropdown.logout")}
							</Button>
						</CardContent>
					</Card>
				</div>
			) : (
				// Lazy pages load inside the shell, so the frame stays while they do.
				<Suspense fallback={<PageSkeleton />}>
					<Outlet />
					<DoctorOnboardingDialog
						enabled={enabledFeatures.clinical !== false}
					/>
				</Suspense>
			)}
		</AppShell>
	);
}
