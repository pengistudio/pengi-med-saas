import { Pencil, Plus, Trash2 } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	deleteSubscription,
	getSubscriptions,
	type Subscription,
} from "@/api/subscription-service";
import { Button } from "@/components/ui/button";
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

const statusColors: Record<string, string> = {
	active: "bg-emerald-500/10 text-emerald-600",
	expired: "bg-red-500/10 text-red-600",
	cancelled: "bg-zinc-500/10 text-zinc-500",
};

const SubscriptionList = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [subscriptions, setSubscriptions] = React.useState<Subscription[]>([]);
	const [loading, setLoading] = React.useState(true);

	const fetch = React.useCallback(async () => {
		setLoading(true);
		const res = await getSubscriptions();
		if (res.success && res.data) setSubscriptions(res.data as Subscription[]);
		setLoading(false);
	}, []);

	React.useEffect(() => {
		fetch();
	}, [fetch]);

	const handleDelete = async (id: number) => {
		const res = await deleteSubscription(id);
		if (res.success) fetch();
	};

	return (
		<DashboardLayout>
			<div className="space-y-6">
				<div className="flex items-center justify-between">
					<h1 className="text-2xl font-bold tracking-tight">
						{textGet("backoffice.subscriptions.title")}
					</h1>
					<Button onClick={() => navigate("/subscriptions/create")}>
						<Plus className="h-4 w-4 mr-2" />
						{textGet("backoffice.subscriptions.create")}
					</Button>
				</div>
				<Card>
					<CardHeader>
						<CardTitle>
							{textGet("backoffice.subscriptions.list.title")}
						</CardTitle>
						<CardDescription>
							{textGet("backoffice.subscriptions.list.description")}
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loading ? (
							<p className="text-sm text-muted-foreground py-8 text-center animate-pulse">
								{textGet("backoffice.companies.loading")}
							</p>
						) : subscriptions.length === 0 ? (
							<p className="text-sm text-muted-foreground py-8 text-center">
								{textGet("backoffice.subscriptions.empty")}
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>
											{textGet("backoffice.subscriptions.col.company")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.subscriptions.col.plan")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.subscriptions.col.status")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.subscriptions.col.expires")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("table.actions")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{subscriptions.map((s) => (
										<TableRow key={s.ID}>
											<TableCell>#{s.CompanyID}</TableCell>
											<TableCell className="font-medium">
												{s.plan?.name ?? s.plan_code}
											</TableCell>
											<TableCell>
												<span
													className={cn(
														"inline-flex items-center rounded-full px-2 py-1 text-xs font-medium",
														statusColors[s.status] ??
															"bg-muted text-muted-foreground",
													)}
												>
													{s.status}
												</span>
											</TableCell>
											<TableCell className="text-muted-foreground">
												{new Date(s.expires_at).toLocaleDateString()}
											</TableCell>
											<TableCell className="text-right">
												<div className="flex items-center justify-end gap-2">
													<Button
														variant="ghost"
														size="icon"
														onClick={() =>
															navigate(`/subscriptions/edit/${s.ID}`)
														}
													>
														<Pencil className="h-4 w-4" />
													</Button>
													<Button
														variant="ghost"
														size="icon"
														onClick={() => handleDelete(s.ID)}
													>
														<Trash2 className="h-4 w-4 text-destructive" />
													</Button>
												</div>
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

export default SubscriptionList;
