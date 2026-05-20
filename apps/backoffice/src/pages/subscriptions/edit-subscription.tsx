import React from "react";
import { useNavigate, useParams } from "react-router";
import z from "zod";
import { getPlans, type Plan, type PricingOption } from "@/api/plan-service";
import {
	getSubscriptions,
	updateSubscription,
} from "@/api/subscription-service";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const formSchema = z.object({ expires_at: z.string().min(1) });

const EditSubscription = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const [loading, setLoading] = React.useState(false);
	const [initialLoading, setInitialLoading] = React.useState(true);
	const [plans, setPlans] = React.useState<Plan[]>([]);
	const [selectedPlan, setSelectedPlan] = React.useState("");
	const [selectedStatus, setSelectedStatus] = React.useState("active");
	const [defaultValues, setDefaultValues] = React.useState({ expires_at: "" });

	React.useEffect(() => {
		getPlans().then((res) => {
			if (res.success && res.data) setPlans(res.data as Plan[]);
		});
	}, []);

	React.useEffect(() => {
		if (!id) return;
		getSubscriptions().then((res) => {
			if (res.success && res.data) {
				const sub = (
					res.data as Array<{
						ID: number;
						plan_code: string;
						status: string;
						expires_at: string;
					}>
				).find((s) => s.ID === Number(id));
				if (sub) {
					setSelectedPlan(sub.plan_code);
					setSelectedStatus(sub.status);
					setDefaultValues({ expires_at: sub.expires_at.split("T")[0] });
				}
			}
			setInitialLoading(false);
		});
	}, [id]);

	const currentPlan = plans.find((p) => p.code === selectedPlan);
	const sortedPricings: PricingOption[] = React.useMemo(() => {
		if (!currentPlan?.pricings?.length) return [];
		return [...currentPlan.pricings].sort((a, b) => a.months - b.months);
	}, [currentPlan]);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!id) return;
		setLoading(true);
		const res = await updateSubscription(id, {
			plan_code: selectedPlan,
			status: selectedStatus,
			expires_at: new Date(values.expires_at).toISOString(),
		});
		setLoading(false);
		if (res.success) navigate("/subscriptions");
	}

	if (initialLoading) {
		return (
			<DashboardLayout>
				<div className="flex items-center justify-center h-64">
					<p className="text-muted-foreground animate-pulse">
						{textGet("backoffice.companies.loading")}
					</p>
				</div>
			</DashboardLayout>
		);
	}

	return (
		<DashboardLayout>
			<div className="max-w-2xl mx-auto">
				<Form<typeof formSchema>
					schema={formSchema}
					onSubmit={onSubmit}
					defaultValues={defaultValues}
				>
					{(field) => (
						<Card>
							<CardHeader>
								<CardTitle>
									{textGet("backoffice.subscriptions.edit.title")}
								</CardTitle>
								<CardDescription>
									{textGet("backoffice.subscriptions.edit.description")}
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-4">
								<div className="space-y-2">
									<Label>{textGet("backoffice.subscriptions.col.plan")}</Label>
									<Select
										value={selectedPlan}
										onValueChange={(v) => v && setSelectedPlan(v)}
									>
										<SelectTrigger>
											<SelectValue />
										</SelectTrigger>
										<SelectContent>
											{plans.map((p) => (
												<SelectItem key={p.ID} value={p.code}>
													{p.name}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
								</div>

								{/* Plan pricing summary — informational only */}
								{sortedPricings.length > 0 && (
									<div className="space-y-2">
										<Label>{textGet("backoffice.plans.pricings.title")}</Label>
										<div className="flex flex-wrap gap-2">
											{sortedPricings.map((p) => (
												<span
													key={p.months}
													className={cn(
														"inline-flex items-center rounded-full px-2.5 py-1 text-xs font-mono",
														"bg-muted text-muted-foreground",
													)}
												>
													{textGet(`subscription.plans.period.${p.months}`)} · $
													{p.price.toFixed(0)}
												</span>
											))}
										</div>
									</div>
								)}

								<div className="space-y-2">
									<Label>
										{textGet("backoffice.subscriptions.col.status")}
									</Label>
									<Select
										value={selectedStatus}
										onValueChange={(v) => v && setSelectedStatus(v)}
									>
										<SelectTrigger>
											<SelectValue />
										</SelectTrigger>
										<SelectContent>
											<SelectItem value="active">Active</SelectItem>
											<SelectItem value="expired">Expired</SelectItem>
											<SelectItem value="cancelled">Cancelled</SelectItem>
										</SelectContent>
									</Select>
								</div>
								<FormInput
									field={field}
									name="expires_at"
									type="date"
									label={textGet("backoffice.subscriptions.col.expires")}
								/>
							</CardContent>
							<CardFooter className="flex justify-between">
								<Button
									type="button"
									variant="outline"
									onClick={() => navigate("/subscriptions")}
								>
									{textGet("backoffice.companies.cancel")}
								</Button>
								<Button type="submit" disabled={loading}>
									{loading && <Spinner />}
									{textGet("backoffice.companies.save")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</DashboardLayout>
	);
};

export default EditSubscription;
