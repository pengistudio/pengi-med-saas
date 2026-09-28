import { useText } from "@pengi/shared";
import {
	Badge,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
	DialogTrigger,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@pengi/ui";
import {
	AlertTriangle,
	Check,
	CheckCircle,
	CreditCard,
	Loader2,
	Lock,
	Minus,
	RotateCcw,
	Shield,
} from "lucide-react";
import React from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import {
	cancelPlanChange,
	confirmPayment,
	getAvailablePlans,
	getMySubscription,
	getSubscriptionPayments,
	initiatePayment,
	type PlanOption,
	type PricingOption,
	type SubscriptionDetail,
	type SubscriptionPaymentRecord,
} from "@/api/subscription-service";
import { PageHeader } from "@/components/custom/page-header";
import { cn } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

// ─── Privacy Modal ────────────────────────────────────────────────────────────

const PRIVACY_POINTS = [
	{
		icon: CreditCard,
		titleKey: "subscription.privacy.modal.no_card_title",
		descKey: "subscription.privacy.modal.no_card_desc",
	},
	{
		icon: Shield,
		titleKey: "subscription.privacy.modal.processor_title",
		descKey: "subscription.privacy.modal.processor_desc",
	},
	{
		icon: Lock,
		titleKey: "subscription.privacy.modal.encryption_title",
		descKey: "subscription.privacy.modal.encryption_desc",
	},
] as const;

function PrivacyModal() {
	const { textGet } = useText();
	return (
		<Dialog>
			<DialogTrigger className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors shrink-0">
				<Shield className="h-3.5 w-3.5" />
				{textGet("subscription.privacy.trigger")}
			</DialogTrigger>
			<DialogContent className="max-w-md">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Shield className="h-5 w-5 text-primary" />
						{textGet("subscription.privacy.modal.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet("subscription.privacy.modal.desc")}
					</DialogDescription>
				</DialogHeader>
				<div className="space-y-4 pt-2">
					{PRIVACY_POINTS.map(({ icon: Icon, titleKey, descKey }) => (
						<div key={titleKey} className="flex gap-3">
							<div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted">
								<Icon className="h-4 w-4 text-muted-foreground" />
							</div>
							<div>
								<p className="text-sm font-medium">{textGet(titleKey)}</p>
								<p className="text-xs text-muted-foreground mt-0.5">
									{textGet(descKey)}
								</p>
							</div>
						</div>
					))}
					<div className="pt-2 border-t">
						<Link
							to="/privacy"
							className="text-xs text-primary hover:underline"
						>
							{textGet("subscription.privacy.modal.full_policy")} →
						</Link>
					</div>
				</div>
			</DialogContent>
		</Dialog>
	);
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const PERIOD_ORDER = [1, 3, 6, 9, 12];
const FEATURE_KEYS = ["clinical", "billing", "team", "kanban"] as const;

/** Renewal opens this many days before the subscription expires. */
const RENEW_WINDOW_DAYS = 30;

function daysColor(days: number) {
	if (days <= 7) return "text-destructive";
	if (days <= RENEW_WINDOW_DAYS) return "text-amber-600 dark:text-amber-400";
	return "text-muted-foreground";
}

function StatusBadge({ status }: { status: string }) {
	const { textGet } = useText();
	const map: Record<
		string,
		{
			label: string;
			variant: "default" | "secondary" | "destructive" | "outline";
		}
	> = {
		active: {
			label: textGet("subscription.status.active"),
			variant: "default",
		},
		expired: {
			label: textGet("subscription.status.expired"),
			variant: "destructive",
		},
		inactive: {
			label: textGet("subscription.status.inactive"),
			variant: "secondary",
		},
		cancelled: {
			label: textGet("subscription.status.cancelled"),
			variant: "outline",
		},
	};
	const cfg = map[status] ?? { label: status, variant: "secondary" as const };
	return <Badge variant={cfg.variant}>{cfg.label}</Badge>;
}

/** Payments that never went through: shown, but quieter than real charges. */
const INACTIVE_PAYMENT_STATUSES = ["cancelled", "expired"];

function PaymentStatus({ status }: { status: string }) {
	const { textGet } = useText();
	const dot: Record<string, string> = {
		paid: "bg-primary",
		pending: "bg-amber-500",
		rejected: "bg-destructive",
	};
	return (
		<span className="inline-flex items-center gap-2 text-sm">
			<span
				className={cn(
					"size-1.5 rounded-full",
					dot[status] ?? "bg-muted-foreground/50",
				)}
			/>
			{textGet(`subscription.payment.status.${status}`)}
		</span>
	);
}

// ─── Plan Card ────────────────────────────────────────────────────────────────

function PlanCard({
	plan,
	currentPlanCode,
	currentPlanTier,
	daysLeft,
	onSelect,
	payingKey,
	pendingChangePlanCode,
	onCancelChange,
	cancellingChange,
}: {
	plan: PlanOption;
	currentPlanCode: string;
	currentPlanTier: number;
	daysLeft: number;
	onSelect: (planCode: string, months: number) => void;
	payingKey: string | null;
	pendingChangePlanCode: string;
	onCancelChange: () => void;
	cancellingChange: boolean;
}) {
	const { textGet, formatMoney } = useText();
	const isCurrent = plan.code === currentPlanCode;
	const isPendingTarget = plan.code === pendingChangePlanCode;
	const hasPendingChange = pendingChangePlanCode !== "";
	const canRenew = isCurrent && daysLeft <= RENEW_WINDOW_DAYS;
	const showButton = !isCurrent || canRenew;

	const tierDiff = isCurrent ? 0 : plan.tier - currentPlanTier;
	const isUpgrade = tierDiff > 0;
	const isDowngrade = tierDiff < 0;
	const isSameTier = !isCurrent && tierDiff === 0;

	const sortedPricings: PricingOption[] =
		plan.pricings && plan.pricings.length > 0
			? [...plan.pricings].sort(
					(a, b) =>
						PERIOD_ORDER.indexOf(a.months) - PERIOD_ORDER.indexOf(b.months),
				)
			: [{ months: 1, price: plan.price }];

	const [selectedMonths, setSelectedMonths] = React.useState(
		sortedPricings[0]?.months ?? 1,
	);

	const selectedPricing =
		sortedPricings.find((p) => p.months === selectedMonths) ??
		sortedPricings[0];
	const perMonth = selectedPricing
		? selectedPricing.price / selectedPricing.months
		: plan.price;
	const isMultiMonth = selectedPricing && selectedPricing.months > 1;
	const anyPaying = payingKey !== null;

	return (
		<Card
			className={cn(
				"flex flex-col",
				isCurrent && "border-primary/50 ring-1 ring-primary/20",
				isPendingTarget && "border-amber-500/60",
			)}
		>
			<CardHeader className="pb-3">
				<div className="flex items-start justify-between gap-2">
					<CardTitle className="text-lg">{plan.name}</CardTitle>
					<div className="flex flex-col items-end gap-1">
						{isCurrent && (
							<Badge variant="default" className="shrink-0">
								{textGet("subscription.plans.current_badge")}
							</Badge>
						)}
						{isPendingTarget && (
							<Badge
								variant="outline"
								className="shrink-0 border-amber-500 text-amber-600 dark:text-amber-400"
							>
								{textGet("subscription.plans.pending_change_badge")}
							</Badge>
						)}
					</div>
				</div>
				{isPendingTarget && (
					<p className="text-xs text-amber-600 dark:text-amber-400 mt-1">
						{textGet("subscription.plans.pending_change_hint")}
					</p>
				)}

				{/* Period selector — only when there is something to pay for */}
				{sortedPricings.length > 1 && showButton && (
					<div className="flex flex-wrap gap-1 pt-1">
						{sortedPricings.map((p) => (
							<button
								key={p.months}
								type="button"
								onClick={() => setSelectedMonths(p.months)}
								className={cn(
									"px-2.5 py-1 rounded-md text-xs font-medium transition-colors",
									selectedMonths === p.months
										? "bg-primary text-primary-foreground"
										: "bg-muted text-muted-foreground hover:bg-muted/80",
								)}
							>
								{textGet(`subscription.plans.period.${p.months}`)}
							</button>
						))}
					</div>
				)}

				{/* Price display */}
				<div className="pt-1">
					<div className="flex items-baseline gap-1">
						<span className="text-3xl font-bold tabular-nums">
							{formatMoney(perMonth)}
						</span>
						<span className="text-sm text-muted-foreground">
							{textGet("subscription.plans.per_month")}
						</span>
					</div>
					{isMultiMonth && (
						<p className="text-xs text-muted-foreground mt-0.5">
							{textGet("subscription.plans.total")}{" "}
							<span className="font-medium text-foreground">
								{formatMoney(selectedPricing.price)}
							</span>
						</p>
					)}
				</div>
			</CardHeader>

			<CardContent className="flex flex-col flex-1 gap-4">
				{/* Feature list */}
				<ul className="space-y-2 flex-1">
					{FEATURE_KEYS.map((feature) => {
						const enabled = plan.enabled_features?.[feature];
						return (
							<li key={feature} className="flex items-center gap-2 text-sm">
								{enabled ? (
									<Check className="h-4 w-4 text-primary shrink-0" />
								) : (
									<Minus className="h-4 w-4 text-muted-foreground/60 shrink-0" />
								)}
								<span
									className={
										enabled
											? "text-foreground"
											: "text-muted-foreground line-through decoration-muted-foreground/40"
									}
								>
									{textGet(`subscription.plans.feature.${feature}`)}
								</span>
							</li>
						);
					})}
				</ul>

				{/* Cancel pending change — shown on the current plan when a deferred change is scheduled */}
				{isCurrent && hasPendingChange && (
					<Button
						className="w-full"
						variant="outline"
						onClick={onCancelChange}
						disabled={cancellingChange || anyPaying}
					>
						{cancellingChange ? (
							<>
								<Loader2 className="h-4 w-4 mr-2 animate-spin" />
								{textGet("subscription.plans.upgrading")}
							</>
						) : (
							<>
								<RotateCcw className="h-4 w-4 mr-2" />
								{textGet("subscription.plans.keep_current_btn")}
							</>
						)}
					</Button>
				)}

				{/* Hidden when: current plan has pending change (handled by cancel button), or pending target not expiring soon */}
				{showButton &&
					!(isCurrent && hasPendingChange) &&
					!(isPendingTarget && daysLeft > 7) &&
					(() => {
						// Downgrade expiring soon (or pending target expiring soon) → pay now at new plan price
						const isExpiringDowngrade =
							(isDowngrade || isPendingTarget) && daysLeft <= 7;
						// Regular downgrade (not expiring) → deferred, pass months=0 so service omits it
						const months =
							isDowngrade && !isExpiringDowngrade ? 0 : selectedMonths;
						const payKey = `${plan.code}:${months}`;
						const isThisPaying = payingKey === payKey;

						return (
							<Button
								className="w-full"
								variant={
									isUpgrade ? "default" : isCurrent ? "default" : "outline"
								}
								onClick={() => onSelect(plan.code, months)}
								disabled={anyPaying}
							>
								{isThisPaying ? (
									<>
										<Loader2 className="h-4 w-4 mr-2 animate-spin" />
										{textGet("subscription.plans.upgrading")}
									</>
								) : isCurrent ? (
									<>
										<CreditCard className="h-4 w-4 mr-2" />
										{textGet("subscription.plans.renew_btn")}
									</>
								) : isUpgrade ? (
									<>
										<CreditCard className="h-4 w-4 mr-2" />
										{textGet("subscription.plans.upgrade_btn")}
									</>
								) : isExpiringDowngrade ? (
									<>
										<CreditCard className="h-4 w-4 mr-2" />
										{textGet("subscription.plans.renew_now_btn")}
									</>
								) : isDowngrade ? (
									<>
										<CreditCard className="h-4 w-4 mr-2" />
										{textGet("subscription.plans.downgrade_btn")}
									</>
								) : isSameTier ? (
									<>
										<CreditCard className="h-4 w-4 mr-2" />
										{textGet("subscription.plans.switch_btn")}
									</>
								) : (
									<>
										<CreditCard className="h-4 w-4 mr-2" />
										{textGet("subscription.plans.select_btn")}
									</>
								)}
							</Button>
						);
					})()}
			</CardContent>
		</Card>
	);
}

// ─── Page ─────────────────────────────────────────────────────────────────────

const MySubscriptionPage = () => {
	const { textGet, formatDate, formatMoney } = useText();
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const paymentSuccess = searchParams.get("status") === "success";

	const [sub, setSub] = React.useState<SubscriptionDetail | null>(null);
	const [payments, setPayments] = React.useState<SubscriptionPaymentRecord[]>(
		[],
	);
	const [plans, setPlans] = React.useState<PlanOption[]>([]);
	const [loading, setLoading] = React.useState(true);
	const [payingKey, setPayingKey] = React.useState<string | null>(null);
	const [cancellingChange, setCancellingChange] = React.useState(false);

	const fetchData = React.useCallback(async () => {
		const [subRes, paymentsRes, plansRes] = await Promise.all([
			getMySubscription(),
			getSubscriptionPayments(),
			getAvailablePlans(),
		]);
		if (subRes.success && subRes.data) setSub(subRes.data);
		if (paymentsRes.success && paymentsRes.data) setPayments(paymentsRes.data);
		if (plansRes.success && plansRes.data) setPlans(plansRes.data);
		setLoading(false);
	}, []);

	React.useEffect(() => {
		fetchData();
	}, [fetchData]);

	React.useEffect(() => {
		if (!paymentSuccess) return;
		navigate("/subscription", { replace: true });
		confirmPayment().then(() => fetchData());
	}, [paymentSuccess, navigate, fetchData]);

	const handlePay = async (planCode: string, months: number) => {
		const key = `${planCode}:${months}`;
		setPayingKey(key);
		const res = await initiatePayment(planCode, months);
		setPayingKey(null);
		if (!res.success) return;
		if (res.data.free) {
			fetchData();
			return;
		}
		if (res.data.checkout_url) {
			window.location.href = res.data.checkout_url;
		}
	};

	const handleCancelChange = async () => {
		setCancellingChange(true);
		const res = await cancelPlanChange();
		setCancellingChange(false);
		if (res.success) fetchData();
	};

	return (
		<DashboardLayout>
			<div className="space-y-6">
				{paymentSuccess && (
					<div className="flex items-center gap-3 rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-4 py-3">
						<CheckCircle className="h-5 w-5 text-emerald-500 shrink-0" />
						<p className="text-sm font-medium text-emerald-700 dark:text-emerald-400">
							{textGet("subscription.payment.success_banner")}
						</p>
					</div>
				)}
				<PageHeader
					title={textGet("subscription.page.title")}
					description={textGet("subscription.page.description")}
				/>

				{/* Subscription overview — informational only */}
				<Card>
					<CardContent>
						{loading ? (
							<div className="space-y-2">
								<div className="h-7 w-40 rounded bg-muted animate-pulse" />
								<div className="h-4 w-72 rounded bg-muted animate-pulse" />
							</div>
						) : sub ? (
							<div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
								<div className="space-y-1">
									<div className="flex flex-wrap items-center gap-2">
										<h2 className="text-2xl font-bold">{sub.plan_name}</h2>
										<StatusBadge status={sub.status} />
									</div>
									<p className="text-sm">
										{textGet("subscription.card.expires_on", {
											date: formatDate(sub.expires_at, "long"),
										})}{" "}
										<span
											className={cn("tabular-nums", daysColor(sub.days_left))}
										>
											(
											{sub.days_left <= 0
												? textGet("subscription.status.expired")
												: `${sub.days_left} ${textGet("subscription.card.days_left")}`}
											)
										</span>
									</p>
									{sub.last_payment_amount > 0 && (
										<p className="text-sm text-muted-foreground tabular-nums">
											{textGet("subscription.card.last_payment")}{" "}
											{textGet(
												`subscription.plans.period.${sub.last_payment_months}`,
											)}
											, {formatMoney(sub.last_payment_amount)}
										</p>
									)}
								</div>
								{sub.days_left <= RENEW_WINDOW_DAYS ? (
									<div className="flex items-start gap-2 text-amber-600 sm:max-w-xs dark:text-amber-400">
										<AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
										<p className="text-sm font-medium">
											{textGet("subscription.card.renew_soon")}
										</p>
									</div>
								) : (
									<p className="text-sm text-muted-foreground sm:max-w-xs sm:text-right">
										{textGet("subscription.card.renew_from", {
											date: formatDate(
												new Date(
													new Date(sub.expires_at).getTime() -
														RENEW_WINDOW_DAYS * 86_400_000,
												),
												"long",
											),
										})}
									</p>
								)}
							</div>
						) : (
							<p className="text-sm text-muted-foreground">—</p>
						)}
					</CardContent>
				</Card>

				{/* Available Plans */}
				{!loading && plans.length > 0 && sub && (
					<div className="space-y-4">
						<div className="flex items-start justify-between gap-4">
							<div>
								<h2 className="text-lg font-semibold">
									{textGet("subscription.plans.section.title")}
								</h2>
								<p className="text-sm text-muted-foreground">
									{textGet("subscription.plans.section.description")}
								</p>
							</div>
							<PrivacyModal />
						</div>
						<div
							className={cn(
								"grid gap-4 sm:grid-cols-2",
								plans.length > 2 && "xl:grid-cols-3",
							)}
						>
							{[...plans]
								.sort((a, b) => {
									if (a.code === sub.plan_code) return -1;
									if (b.code === sub.plan_code) return 1;
									return 0;
								})
								.map((plan) => (
									<PlanCard
										key={plan.code}
										plan={plan}
										currentPlanCode={sub.plan_code}
										currentPlanTier={sub.plan_tier}
										daysLeft={sub.days_left}
										onSelect={handlePay}
										payingKey={payingKey}
										pendingChangePlanCode={sub.next_plan_code ?? ""}
										onCancelChange={handleCancelChange}
										cancellingChange={cancellingChange}
									/>
								))}
						</div>
					</div>
				)}

				{/* Payment History */}
				<Card>
					<CardHeader>
						<CardTitle>{textGet("subscription.payments.title")}</CardTitle>
						<CardDescription>
							{textGet("subscription.payments.description")}
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loading ? (
							<div className="space-y-2">
								{[1, 2, 3].map((i) => (
									<div
										key={i}
										className="h-10 w-full bg-muted animate-pulse rounded"
									/>
								))}
							</div>
						) : payments.length === 0 ? (
							<p className="text-sm text-muted-foreground text-center py-8">
								{textGet("subscription.payments.empty")}
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>
											{textGet("subscription.payments.col.date")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("subscription.payments.col.amount")}
										</TableHead>
										<TableHead>
											{textGet("subscription.payments.col.status")}
										</TableHead>
										<TableHead className="hidden sm:table-cell">
											{textGet("subscription.payments.col.order")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{payments.map((p) => {
										const inactive = INACTIVE_PAYMENT_STATUSES.includes(
											p.status,
										);
										return (
											<TableRow
												key={p.ID}
												className={cn(inactive && "text-muted-foreground")}
											>
												<TableCell className="text-sm">
													{formatDate(p.CreatedAt)}
												</TableCell>
												<TableCell
													className={cn(
														"text-right text-sm tabular-nums",
														!inactive && "font-medium",
													)}
												>
													{formatMoney(p.amount)}
												</TableCell>
												<TableCell>
													<PaymentStatus status={p.status} />
												</TableCell>
												<TableCell className="hidden text-xs text-muted-foreground font-mono sm:table-cell">
													{p.order_id.slice(0, 8)}
												</TableCell>
											</TableRow>
										);
									})}
								</TableBody>
							</Table>
						)}
					</CardContent>
				</Card>
			</div>
		</DashboardLayout>
	);
};

export default MySubscriptionPage;
