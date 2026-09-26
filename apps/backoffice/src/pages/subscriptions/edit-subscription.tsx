import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	FormInput,
	Label,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
	Spinner,
} from "@pengi/ui";
import React from "react";
import { useNavigate, useParams } from "react-router";
import z from "zod";
import {
	type Plan,
	type PricingOption,
	plans as planResource,
} from "@/api/plan-service";
import { subscriptions } from "@/api/subscription-service";
import { Form } from "@/components/forms/form";
import { useText } from "@/hooks/use-text";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";
import { cn } from "@/lib/utils";

const formSchema = z.object({ expires_at: z.string().min(1) });

const EditSubscription = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const {
		item: subscription,
		loading,
		saving,
		save,
	} = useResourceItem(subscriptions, id);
	const [plans, setPlans] = React.useState<Plan[]>([]);
	const [selectedPlan, setSelectedPlan] = React.useState("");
	const [selectedStatus, setSelectedStatus] = React.useState("active");
	const defaultValues = {
		expires_at: subscription?.expires_at.split("T")[0] ?? "",
	};

	React.useEffect(() => {
		planResource.list().then((res) => {
			if (res.success && res.data) setPlans(res.data as Plan[]);
		});
	}, []);

	React.useEffect(() => {
		if (!subscription) return;
		setSelectedPlan(subscription.plan_code);
		setSelectedStatus(subscription.status);
	}, [subscription]);

	const currentPlan = plans.find((p) => p.code === selectedPlan);
	const sortedPricings: PricingOption[] = React.useMemo(() => {
		if (!currentPlan?.pricings?.length) return [];
		return [...currentPlan.pricings].sort((a, b) => a.months - b.months);
	}, [currentPlan]);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		await save({
			plan_code: selectedPlan,
			status: selectedStatus,
			expires_at: new Date(values.expires_at).toISOString(),
		});
	}

	return (
		<ResourceEditPage loading={loading}>
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
									{textGet("backoffice.common.cancel")}
								</Button>
								<Button type="submit" disabled={saving}>
									{saving && <Spinner />}
									{textGet("backoffice.common.save")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</ResourceEditPage>
	);
};

export default EditSubscription;
