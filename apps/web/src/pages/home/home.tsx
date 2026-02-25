import {
	AlertTriangle,
	Calendar,
	CheckCircle,
	Clock,
	Users,
} from "lucide-react";
import React from "react";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { type DashboardStats, getDashboardStats } from "@/api/clinical-service";
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

function StatCard({
	title,
	value,
	icon: Icon,
	iconClassName,
}: {
	title: string;
	value: number | string;
	icon: React.ComponentType<{ className?: string }>;
	iconClassName?: string;
}) {
	return (
		<Card>
			<CardHeader className="flex flex-row items-center justify-between pb-2">
				<CardDescription className="text-sm font-medium">
					{title}
				</CardDescription>
				<Icon className={cn("h-5 w-5 text-muted-foreground", iconClassName)} />
			</CardHeader>
			<CardContent>
				<p className="text-3xl font-bold tracking-tight">{value}</p>
			</CardContent>
		</Card>
	);
}

// ─── Home ────────────────────────────────────────────────────────────────────

const Home = () => {
	const [stats, setStats] = React.useState<DashboardStats | null>(null);
	const { textGet } = useText();

	React.useEffect(() => {
		getDashboardStats().then((res) => {
			if (res.success && res.data) {
				setStats(res.data as DashboardStats);
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

	return (
		<DashboardLayout>
			<div className="space-y-6">
				{/* Header */}
				<h1 className="text-2xl font-bold tracking-tight">
					<Text uuid="dashboard.title" />
				</h1>

				{/* Stat Cards */}
				<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
					<StatCard
						title={textGet("dashboard.stat.total_patients")}
						value={stats.total_patients}
						icon={Users}
					/>
					<StatCard
						title={textGet("dashboard.stat.critical_patients")}
						value={stats.critical_patients}
						icon={AlertTriangle}
						iconClassName="text-red-500"
					/>
					<StatCard
						title={textGet("dashboard.stat.today_appointments")}
						value={stats.today_appointments}
						icon={Calendar}
						iconClassName="text-blue-500"
					/>
					<StatCard
						title={textGet("dashboard.stat.monthly_completed")}
						value={stats.monthly_completed}
						icon={CheckCircle}
						iconClassName="text-emerald-500"
					/>
				</div>

				{/* Charts Row */}
				<div className="grid gap-4 lg:grid-cols-5">
					{/* Weekly Bar Chart — takes 3 columns */}
					<Card className="lg:col-span-3">
						<CardHeader>
							<CardTitle>
								<Text uuid="dashboard.chart.weekly_title" />
							</CardTitle>
							<CardDescription>
								<Text uuid="dashboard.chart.weekly_description" />
							</CardDescription>
						</CardHeader>
						<CardContent>
							<ChartContainer config={chartConfig} className="h-[260px] w-full">
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
										width={30}
									/>
									<ChartTooltip
										content={<ChartTooltipContent />}
										cursor={false}
									/>
									<Bar
										dataKey="count"
										fill="var(--color-count)"
										radius={[6, 6, 0, 0]}
									/>
								</BarChart>
							</ChartContainer>
						</CardContent>
					</Card>

					{/* Upcoming Appointments — takes 2 columns */}
					<Card className="lg:col-span-2">
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
								<p className="text-sm text-muted-foreground py-8 text-center">
									{textGet("dashboard.upcoming.empty")}
								</p>
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
