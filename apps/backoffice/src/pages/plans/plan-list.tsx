import { Pencil, Plus, Trash2 } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import { deletePlan, getPlans, type Plan } from "@/api/plan-service";
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
import { DashboardLayout } from "@/sections/template/dashboard-template";

const PlanList = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [plans, setPlans] = React.useState<Plan[]>([]);
	const [loading, setLoading] = React.useState(true);

	const fetch = React.useCallback(async () => {
		setLoading(true);
		const res = await getPlans();
		if (res.success && res.data) setPlans(res.data as Plan[]);
		setLoading(false);
	}, []);

	React.useEffect(() => {
		fetch();
	}, [fetch]);

	const handleDelete = async (id: number) => {
		const res = await deletePlan(id);
		if (res.success) fetch();
	};

	return (
		<DashboardLayout>
			<div className="space-y-6">
				<div className="flex items-center justify-between">
					<h1 className="text-2xl font-bold tracking-tight">
						{textGet("backoffice.plans.title")}
					</h1>
					<Button onClick={() => navigate("/plans/create")}>
						<Plus className="h-4 w-4 mr-2" />
						{textGet("backoffice.plans.create")}
					</Button>
				</div>
				<Card>
					<CardHeader>
						<CardTitle>{textGet("backoffice.plans.list.title")}</CardTitle>
						<CardDescription>
							{textGet("backoffice.plans.list.description")}
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loading ? (
							<p className="text-sm text-muted-foreground py-8 text-center animate-pulse">
								{textGet("backoffice.companies.loading")}
							</p>
						) : plans.length === 0 ? (
							<p className="text-sm text-muted-foreground py-8 text-center">
								{textGet("backoffice.plans.empty")}
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>
											{textGet("backoffice.plans.col.name")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.plans.col.code")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.plans.col.tier")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.plans.pricings.title")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.plans.col.features")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("table.actions")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{plans.map((p) => (
										<TableRow key={p.ID}>
											<TableCell className="font-medium">{p.name}</TableCell>
											<TableCell className="font-mono text-sm">
												{p.code}
											</TableCell>
											<TableCell>
												<span className="inline-flex items-center justify-center w-7 h-7 rounded-full bg-muted text-xs font-bold">
													{p.tier ?? 1}
												</span>
											</TableCell>
											<TableCell>
												{p.pricings && p.pricings.length > 0 ? (
													<div className="flex flex-wrap gap-1">
														{[...p.pricings]
															.sort((a, b) => a.months - b.months)
															.map((pr) => (
																<span
																	key={pr.months}
																	className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs font-mono"
																>
																	{pr.months}m · ${pr.price.toFixed(0)}
																</span>
															))}
													</div>
												) : (
													<span className="text-sm text-muted-foreground">
														${p.price.toFixed(2)}/mes
													</span>
												)}
											</TableCell>
											<TableCell>
												<span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
													{p.Features?.length ?? 0}
												</span>
											</TableCell>
											<TableCell className="text-right">
												<div className="flex items-center justify-end gap-2">
													<Button
														variant="ghost"
														size="icon"
														onClick={() => navigate(`/plans/edit/${p.ID}`)}
													>
														<Pencil className="h-4 w-4" />
													</Button>
													<Button
														variant="ghost"
														size="icon"
														onClick={() => handleDelete(p.ID)}
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

export default PlanList;
