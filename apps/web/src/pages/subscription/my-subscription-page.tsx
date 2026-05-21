import {
	AlertTriangle,
	CheckCircle,
	CreditCard,
	Loader2,
	Lock,
	RotateCcw,
	Shield,
	XCircle,
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
	DialogTrigger,
} from "@/components/ui/dialog";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { useText } from "@/hooks/use-text";
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

function daysColor(days: number) {
	if (days <= 7) return "text-red-500";
	if (days <= 15) return "text-amber-500";
	return "text-emerald-500";
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

function PaymentStatusBadge({ status }: { status: string }) {
	const { textGet } = useText();
	const icons: Record<string, React.ReactNode> = {
		pending: <Loader2 className="h-3 w-3 animate-spin" />,
		paid: <CheckCircle className="h-3 w-3" />,
		rejected: <XCircle className="h-3 w-3" />,
		cancelled: <XCircle className="h-3 w-3" />,
		expired: <AlertTriangle className="h-3 w-3" />,
	};
	const variants: Record<
		string,
		"default" | "secondary" | "destructive" | "outline"
	> = {
		paid: "default",
		pending: "secondary",
		rejected: "destructive",
		cancelled: "outline",
		expired: "outline",
	};
	return (
		<Badge variant={variants[status] ?? "secondary"} className="gap-1">
			{icons[status]}
			{textGet(`subscription.payment.status.${status}`)}
		</Badge>
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
	const { textGet } = useText();
	const isCurrent = plan.code === currentPlanCode;
	const isPendingTarget = plan.code === pendingChangePlanCode;
	const hasPendingChange = pendingChangePlanCode !== "";
	const canRenew = isCurrent && daysLeft <= 30;
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
				isCurrent ? "border-t-2 border-t-primary" : "border-border",
				isPendingTarget ? "border-t-2 border-t-amber-500" : "",
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

				{/* Period selector — always shown when multiple periods exist */}
				{sortedPricings.length > 1 && (
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
						<span className="text-3xl font-bold">${perMonth.toFixed(2)}</span>
						<span className="text-sm text-muted-foreground">
							{textGet("subscription.plans.per_month")}
						</span>
					</div>
					{isMultiMonth && (
						<p className="text-xs text-muted-foreground mt-0.5">
							{textGet("subscription.plans.total")}{" "}
							<span className="font-medium text-foreground">
								${selectedPricing.price.toFixed(2)}
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
									<CheckCircle className="h-4 w-4 text-emerald-500 shrink-0" />
								) : (
									<XCircle className="h-4 w-4 text-muted-foreground shrink-0" />
								)}
								<span
									className={
										enabled ? "text-foreground" : "text-muted-foreground"
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
	const { textGet } = useText();
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
				<div>
					<h1 className="text-2xl font-bold tracking-tight">
						{textGet("subscription.page.title")}
					</h1>
					<p className="text-muted-foreground text-sm mt-1">
						{textGet("subscription.page.description")}
					</p>
				</div>

				{/* Subscription overview — informational only */}
				<Card className="border-t-2 border-t-primary">
					<CardContent className="pt-2">
						{loading ? (
							<div className="flex gap-8">
								{[1, 2, 3].map((i) => (
									<div
										key={i}
										className="h-12 w-32 bg-muted animate-pulse rounded"
									/>
								))}
							</div>
						) : sub ? (
							<div className="grid grid-cols-2 gap-6 sm:grid-cols-3 sm:divide-x sm:gap-0 items-center">
								<div className="sm:pr-6 flex flex-col gap-1">
									<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
										{textGet("subscription.card.plan")}
									</p>
									<div className="flex items-center gap-2 flex-wrap">
										<p className="text-2xl font-bold">{sub.plan_name}</p>
										<StatusBadge status={sub.status} />
									</div>
								</div>
								<div className="sm:px-6 flex flex-col gap-1">
									<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
										{textGet("subscription.card.expiry")}
									</p>
									<p className="text-2xl font-bold">
										{new Date(sub.expires_at).toLocaleDateString()}
									</p>
									<p
										className={cn(
											"text-xs font-medium",
											daysColor(sub.days_left),
										)}
									>
										{sub.days_left <= 0
											? textGet("subscription.status.expired")
											: `${sub.days_left} ${textGet("subscription.card.days_left")}`}
									</p>
									{sub.last_payment_amount > 0 && (
										<p className="text-xs text-muted-foreground mt-0.5">
											{textGet(
												`subscription.plans.period.${sub.last_payment_months}`,
											)}{" "}
											· ${sub.last_payment_amount.toFixed(2)}
										</p>
									)}
								</div>
								<div className="sm:pl-6 flex items-center">
									{sub.days_left <= 30 ? (
										<div className="flex items-start gap-2 text-amber-600 dark:text-amber-400">
											<AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
											<p className="text-sm font-medium">
												{textGet("subscription.card.renew_soon")}
											</p>
										</div>
									) : (
										<p className="text-sm text-muted-foreground text-center">
											{textGet("subscription.card.renew_hint")}
										</p>
									)}
								</div>
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
						<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
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
										<TableHead>
											{textGet("subscription.payments.col.amount")}
										</TableHead>
										<TableHead>
											{textGet("subscription.payments.col.status")}
										</TableHead>
										<TableHead>
											{textGet("subscription.payments.col.order")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{payments.map((p) => (
										<TableRow key={p.ID}>
											<TableCell className="text-sm">
												{new Date(p.CreatedAt).toLocaleDateString()}
											</TableCell>
											<TableCell className="text-sm font-medium">
												${p.amount.toFixed(2)}
											</TableCell>
											<TableCell>
												<PaymentStatusBadge status={p.status} />
											</TableCell>
											<TableCell className="text-xs text-muted-foreground font-mono">
												{p.order_id.slice(0, 8)}…
											</TableCell>
										</TableRow>
									))}
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
