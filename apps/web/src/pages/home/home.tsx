import {
	AlertTriangle,
	ArrowRight,
	Calendar,
	CalendarPlus,
	CheckCircle,
	Clock,
	Users,
} from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { getAllInvoices, type Invoice } from "@/api/billing-service";
import {
	type DashboardStats,
	getDashboardStats,
	type SubscriptionInfo,
} from "@/api/clinical-service";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	type ChartConfig,
	ChartContainer,
	ChartTooltip,
	ChartTooltipContent,
} from "@/components/ui/chart";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

// ─── Chart Config ────────────────────────────────────────────────────────────

const chartConfig = {
	count: {
		label: "Citas",
		color: "var(--color-primary)",
	},
} satisfies ChartConfig;

// ─── Stat Card ───────────────────────────────────────────────────────────────

type StatAccent = "default" | "red" | "blue" | "emerald";

const ACCENT_STYLES: Record<
	StatAccent,
	{ iconBg: string; iconText: string; card: string }
> = {
	default: { iconBg: "bg-muted", iconText: "text-muted-foreground", card: "" },
	red: {
		iconBg: "bg-red-500/10",
		iconText: "text-red-500",
		card: "border-red-500/40 bg-red-500/5",
	},
	blue: { iconBg: "bg-blue-500/10", iconText: "text-blue-500", card: "" },
	emerald: {
		iconBg: "bg-emerald-500/10",
		iconText: "text-emerald-500",
		card: "",
	},
};

function DeltaBadge({ delta, label }: { delta: number; label: string }) {
	if (delta === 0) return null;
	const positive = delta > 0;
	return (
		<p
			className={cn(
				"text-xs font-medium",
				positive ? "text-emerald-500" : "text-red-500",
			)}
		>
			{positive ? "↑" : "↓"} {Math.abs(delta)} {label}
		</p>
	);
}

function StatCard({
	title,
	value,
	icon: Icon,
	accent = "default",
	alertWhenPositive = false,
	delta,
	deltaLabel,
}: {
	title: string;
	value: number | string;
	icon: React.ComponentType<{ className?: string }>;
	accent?: StatAccent;
	alertWhenPositive?: boolean;
	delta?: number;
	deltaLabel?: string;
}) {
	const isAlert = alertWhenPositive && typeof value === "number" && value > 0;
	const styles = ACCENT_STYLES[isAlert ? "red" : accent];

	return (
		<Card className={styles.card}>
			<CardContent className="pt-5 pb-5">
				<div className="flex items-start justify-between gap-3">
					<div className="space-y-1">
						<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
							{title}
						</p>
						<p className="text-3xl font-bold tracking-tight">{value}</p>
						{delta !== undefined && deltaLabel && (
							<DeltaBadge delta={delta} label={deltaLabel} />
						)}
					</div>
					<div className={cn("p-2.5 rounded-xl shrink-0", styles.iconBg)}>
						<Icon className={cn("h-5 w-5", styles.iconText)} />
					</div>
				</div>
			</CardContent>
		</Card>
	);
}

// ─── Subscription Card ───────────────────────────────────────────────────────

function SubscriptionCard({
	subscription,
}: {
	subscription: SubscriptionInfo;
}) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const isActive = subscription.status === "active";
	const isExpiringSoon = subscription.days_left <= 30;
	const isUrgent = subscription.days_left <= 7;

	const progressPct = Math.min(
		100,
		Math.round((subscription.days_left / 365) * 100),
	);
	const progressColor = isUrgent
		? "bg-red-500"
		: isExpiringSoon
			? "bg-amber-400"
			: "bg-emerald-500";

	const headerGradient = !isActive
		? "from-zinc-600 to-zinc-500"
		: isUrgent
			? "from-red-600 to-rose-500"
			: isExpiringSoon
				? "from-amber-500 to-orange-400"
				: "from-teal-600 to-emerald-500";

	// Build subtitle: prefer last payment info, fall back to plan.Price
	const hasLastPayment = subscription.last_payment_amount > 0;
	const subtitleParts: string[] = [textGet("dashboard.subscription.title")];
	if (hasLastPayment) {
		const months = subscription.last_payment_months || 1;
		subtitleParts.push(
			` · $${subscription.last_payment_amount.toFixed(0)} · ${textGet(`subscription.plans.period.${months}`)}`,
		);
	} else if (subscription.amount > 0) {
		subtitleParts.push(` · $${subscription.amount.toFixed(0)} /mes`);
	}

	return (
		<Card className="overflow-hidden flex flex-col h-full py-0">
			{/* Gradient header */}
			<div className={cn("bg-linear-to-r px-5 py-4", headerGradient)}>
				<div className="flex items-start justify-between gap-2">
					<p className="text-xl font-bold text-white leading-tight">
						{subscription.plan_name}
					</p>
					<span className="shrink-0 rounded-full bg-white/20 px-3 py-0.5 text-xs font-medium text-white backdrop-blur-sm">
						{textGet(`subscription.status.${subscription.status}`)}
					</span>
				</div>
				<p className="mt-1 text-sm text-white/80">{subtitleParts.join("")}</p>
			</div>

			{/* Body */}
			<CardContent className="flex flex-col flex-1 gap-3 pt-3 pb-4">
				<div className="flex items-start justify-between">
					<div>
						<p className="text-3xl font-bold leading-none">
							{subscription.days_left}
						</p>
						<p className="text-sm text-muted-foreground mt-0.5">
							{textGet("dashboard.subscription.days_left")}
						</p>
					</div>
					<div className="text-right">
						<p className="text-xs text-muted-foreground">
							{textGet("dashboard.subscription.expires")}
						</p>
						<p className="text-sm font-bold">
							{new Date(subscription.expires_at).toLocaleDateString(undefined, {
								day: "numeric",
								month: "short",
								year: "numeric",
							})}
						</p>
					</div>
				</div>

				{/* Progress bar */}
				<div className="h-1.5 rounded-full bg-muted overflow-hidden">
					<div
						className={cn("h-full rounded-full transition-all", progressColor)}
						style={{ width: `${progressPct}%` }}
					/>
				</div>

				<button
					type="button"
					onClick={() => navigate("/subscription")}
					className="text-sm font-medium text-primary hover:underline flex items-center gap-1 mt-auto"
				>
					{textGet("dashboard.subscription.view_details")}
					<ArrowRight className="h-3.5 w-3.5" />
				</button>
			</CardContent>
		</Card>
	);
}

// ─── Invoice Status Badge ────────────────────────────────────────────────────

const INVOICE_STATUS_STYLES: Record<string, { className: string }> = {
	authorized: { className: "bg-emerald-500/10 text-emerald-600" },
	validated: { className: "bg-emerald-500/10 text-emerald-600" },
	pending: { className: "bg-amber-500/10 text-amber-600" },
	processing: { className: "bg-blue-500/10 text-blue-600" },
	signed: { className: "bg-blue-500/10 text-blue-600" },
};

function InvoiceStatusBadge({ status }: { status: string }) {
	const { textGet } = useText();
	const style = INVOICE_STATUS_STYLES[status] ?? {
		className: "bg-muted text-muted-foreground",
	};
	return (
		<span
			className={cn(
				"inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
				style.className,
			)}
		>
			{textGet(`billing.status.${status}`)}
		</span>
	);
}

// ─── Recent Invoices Card ────────────────────────────────────────────────────

function RecentInvoicesCard({
	invoices,
	onViewAll,
}: {
	invoices: Invoice[];
	onViewAll: () => void;
}) {
	const { textGet } = useText();
	return (
		<Card>
			<CardHeader className="flex flex-row items-center justify-between pb-3">
				<CardTitle className="text-sm font-semibold">
					{textGet("dashboard.recent_invoices.title")}
				</CardTitle>
				<button
					type="button"
					onClick={onViewAll}
					className="text-xs font-medium text-primary hover:underline flex items-center gap-1"
				>
					{textGet("dashboard.recent_invoices.view_all")}
					<ArrowRight className="h-3 w-3" />
				</button>
			</CardHeader>
			<CardContent className="pt-0">
				{invoices.length === 0 ? (
					<p className="text-sm text-muted-foreground py-4 text-center">
						{textGet("dashboard.recent_invoices.empty")}
					</p>
				) : (
					<div className="divide-y">
						{invoices.map((invoice) => {
							const patientName = invoice.patient
								? `${invoice.patient.first_name} ${invoice.patient.last_name}`
								: `#${invoice.sequential}`;
							return (
								<div
									key={invoice.ID}
									className="flex items-center justify-between gap-3 py-2.5"
								>
									<span className="text-sm text-foreground truncate flex-1">
										{patientName}
									</span>
									<span className="text-sm font-semibold tabular-nums">
										${invoice.total.toFixed(2)}
									</span>
									<InvoiceStatusBadge status={invoice.status} />
								</div>
							);
						})}
					</div>
				)}
			</CardContent>
		</Card>
	);
}

// ─── Home ────────────────────────────────────────────────────────────────────

const Home = () => {
	const [stats, setStats] = React.useState<DashboardStats | null>(null);
	const [recentInvoices, setRecentInvoices] = React.useState<Invoice[]>([]);
	const { textGet } = useText();
	const navigate = useNavigate();

	React.useEffect(() => {
		getDashboardStats().then((res) => {
			if (res.success && res.data) {
				const data = res.data as DashboardStats;
				setStats(data);
				if (data.subscription?.enabled_features?.billing !== false) {
					getAllInvoices({ limit: 5 }).then((inv) => {
						if (inv.success && inv.data) setRecentInvoices(inv.data.items);
					});
				}
			}
		});
	}, []);

	if (!stats) {
		return (
			<DashboardLayout>
				<div className="flex items-center justify-center h-64">
					<p className="text-muted-foreground animate-pulse">
						{textGet("dashboard.loading")}
					</p>
				</div>
			</DashboardLayout>
		);
	}

	const yMax = Math.max(...stats.weekly_appointments.map((d) => d.count), 5);

	return (
		<DashboardLayout>
			<div className="space-y-6">
				{/* Stat Cards */}
				<div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4">
					<StatCard
						title={textGet("dashboard.stat.total_patients")}
						value={stats.total_patients}
						icon={Users}
						accent="default"
						delta={stats.new_patients_this_month}
						deltaLabel={textGet("dashboard.stat.delta.new_this_month")}
					/>
					<StatCard
						title={textGet("dashboard.stat.critical_patients")}
						value={stats.critical_patients}
						icon={AlertTriangle}
						alertWhenPositive
					/>
					<StatCard
						title={textGet("dashboard.stat.today_appointments")}
						value={stats.today_appointments}
						icon={Calendar}
						accent="blue"
						delta={stats.today_appointments - stats.yesterday_appointments}
						deltaLabel={textGet("dashboard.stat.delta.vs_yesterday")}
					/>
					<StatCard
						title={textGet("dashboard.stat.monthly_completed")}
						value={stats.monthly_completed}
						icon={CheckCircle}
						accent="emerald"
						delta={stats.monthly_completed - stats.prev_month_completed}
						deltaLabel={textGet("dashboard.stat.delta.vs_last_month")}
					/>
				</div>

				{/* Charts Row */}
				<div className="grid gap-4 xl:grid-cols-5">
					{/* Weekly Bar Chart */}
					<Card className="xl:col-span-3">
						<CardHeader>
							<CardTitle>
								<Text uuid="dashboard.chart.weekly_title" />
							</CardTitle>
							<CardDescription>
								<Text uuid="dashboard.chart.weekly_description" />
							</CardDescription>
						</CardHeader>
						<CardContent>
							<ChartContainer config={chartConfig} className="h-[220px] w-full">
								<BarChart data={stats.weekly_appointments} accessibilityLayer>
									<CartesianGrid vertical={false} />
									<XAxis
										dataKey="day"
										tickLine={false}
										axisLine={false}
										tickMargin={8}
									/>
									<YAxis
										tickLine={false}
										axisLine={false}
										allowDecimals={false}
										width={24}
										domain={[0, yMax]}
									/>
									<ChartTooltip
										content={<ChartTooltipContent />}
										cursor={false}
									/>
									<Bar
										dataKey="count"
										fill="var(--color-count)"
										radius={[6, 6, 0, 0]}
										minPointSize={4}
									/>
								</BarChart>
							</ChartContainer>
						</CardContent>
					</Card>

					{/* Upcoming Appointments */}
					<Card className="xl:col-span-2">
						<CardHeader>
							<CardTitle>
								<Text uuid="dashboard.upcoming.title" />
							</CardTitle>
							<CardDescription>
								<Text uuid="dashboard.upcoming.description" />
							</CardDescription>
						</CardHeader>
						<CardContent>
							{stats.upcoming_appointments.length === 0 ? (
								<div className="flex flex-col items-center justify-center gap-3 py-10 text-center">
									<div className="p-3 rounded-full bg-muted">
										<CalendarPlus className="h-6 w-6 text-muted-foreground" />
									</div>
									<div className="space-y-1">
										<p className="text-sm font-medium">
											{textGet("dashboard.upcoming.empty")}
										</p>
										<p className="text-xs text-muted-foreground">
											{textGet("dashboard.upcoming.empty_hint")}
										</p>
									</div>
									<Button
										size="sm"
										variant="outline"
										onClick={() => navigate("/clinical/appointments")}
									>
										<CalendarPlus className="h-4 w-4 mr-2" />
										{textGet("dashboard.upcoming.schedule_btn")}
									</Button>
								</div>
							) : (
								<div className="space-y-3">
									{stats.upcoming_appointments.map((appt) => (
										<div
											key={appt.id}
											className="flex items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-muted/50"
										>
											<div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
												<Clock className="h-4 w-4" />
											</div>
											<div className="flex-1 min-w-0">
												<p className="text-sm font-medium truncate">
													{appt.title}
												</p>
												<p className="text-xs text-muted-foreground truncate">
													{appt.patient_name}
												</p>
											</div>
											<span className="text-xs font-medium text-muted-foreground whitespace-nowrap">
												{appt.start_time} — {appt.end_time}
											</span>
										</div>
									))}
								</div>
							)}
						</CardContent>
					</Card>
				</div>

				{/* Bottom Row — Subscription + Recent Invoices */}
				{stats.subscription && (
					<div
						className={
							stats.subscription.enabled_features?.billing !== false
								? "grid gap-4 xl:grid-cols-3"
								: ""
						}
					>
						<SubscriptionCard subscription={stats.subscription} />
						{stats.subscription.enabled_features?.billing !== false && (
							<div className="xl:col-span-2">
								<RecentInvoicesCard
									invoices={recentInvoices}
									onViewAll={() => navigate("/billing")}
								/>
							</div>
						)}
					</div>
				)}
			</div>
		</DashboardLayout>
	);
};

export default Home;
