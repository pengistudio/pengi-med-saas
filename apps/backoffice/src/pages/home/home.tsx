import { Building2, CreditCard, Puzzle, Shield, Users } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	type DashboardStats,
	getDashboardStats,
} from "@/api/dashboard-service";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
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

const Home = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [stats, setStats] = React.useState<DashboardStats | null>(null);
	const [loading, setLoading] = React.useState(true);

	React.useEffect(() => {
		getDashboardStats().then((res) => {
			if (res.success && res.data) setStats(res.data as DashboardStats);
			setLoading(false);
		});
	}, []);

	return (
		<DashboardLayout>
			<div className="space-y-6">
				{/* Header */}
				<h1 className="text-2xl font-bold tracking-tight">
					{textGet("backoffice.dashboard.title")}
				</h1>

				{/* Stat Cards */}
				<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
					<StatCard
						title={textGet("backoffice.dashboard.stat.total_companies")}
						value={loading ? "—" : (stats?.total_companies ?? 0)}
						icon={Building2}
					/>
					<StatCard
						title={textGet("backoffice.dashboard.stat.total_users")}
						value={loading ? "—" : (stats?.total_users ?? 0)}
						icon={Users}
						iconClassName="text-blue-500"
					/>
					<StatCard
						title={textGet("backoffice.dashboard.stat.active_plans")}
						value={loading ? "—" : (stats?.total_plans ?? 0)}
						icon={Shield}
						iconClassName="text-emerald-500"
					/>
					<StatCard
						title={textGet("backoffice.dashboard.stat.total_features")}
						value={loading ? "—" : (stats?.total_features ?? 0)}
						icon={Puzzle}
						iconClassName="text-purple-500"
					/>
					<StatCard
						title={textGet("backoffice.dashboard.stat.active_subscriptions")}
						value={loading ? "—" : (stats?.active_subscriptions ?? 0)}
						icon={CreditCard}
						iconClassName="text-amber-500"
					/>
				</div>

				{/* Recent Companies */}
				<Card>
					<CardHeader>
						<CardTitle>
							{textGet("backoffice.dashboard.recent.title")}
						</CardTitle>
						<CardDescription>
							{textGet("backoffice.dashboard.recent.description")}
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loading ? (
							<p className="text-sm text-muted-foreground py-8 text-center animate-pulse">
								{textGet("backoffice.companies.loading")}
							</p>
						) : !stats?.recent_companies?.length ? (
							<p className="text-sm text-muted-foreground py-8 text-center">
								{textGet("backoffice.dashboard.recent.empty")}
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>
											{textGet("backoffice.companies.col.trade_name")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.companies.col.plan")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.companies.col.tenant")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{stats.recent_companies.map((c) => (
										<TableRow
											key={c.ID}
											className="cursor-pointer hover:bg-muted/50"
											onClick={() => navigate(`/companies/edit/${c.ID}`)}
										>
											<TableCell className="font-medium">
												{c.trade_name}
											</TableCell>
											<TableCell>
												<span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
													{c.plan_code}
												</span>
											</TableCell>
											<TableCell className="text-muted-foreground">
												{c.tenant?.slug}
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

export default Home;
