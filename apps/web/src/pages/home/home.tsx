import {
	AlertTriangle,
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
import { type DashboardStats, getDashboardStats } from "@/api/clinical-service";
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

// ─── Home ────────────────────────────────────────────────────────────────────

const Home = () => {
	const [stats, setStats] = React.useState<DashboardStats | null>(null);
	const [recentInvoices, setRecentInvoices] = React.useState<Invoice[]>([]); // used in Task 3
	void recentInvoices;
	const { textGet } = useText();
	const navigate = useNavigate();

	React.useEffect(() => {
		getDashboardStats().then((res) => {
			if (res.success && res.data) {
				setStats(res.data as DashboardStats);
			}
		});
		getAllInvoices({ limit: 5 }).then((res) => {
			if (res.success && res.data) {
				setRecentInvoices(res.data.items.slice(0, 5));
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
			</div>
		</DashboardLayout>
	);
};

export default Home;
