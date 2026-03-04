import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import { type Company, getCompanies } from "@/api/company-service";
import { getPlans, type Plan } from "@/api/plan-service";
import { createSubscription } from "@/api/subscription-service";
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
import { DashboardLayout } from "@/sections/template/dashboard-template";

const formSchema = z.object({ expires_at: z.string().min(1) });

const CreateSubscription = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [loading, setLoading] = React.useState(false);
	const [companies, setCompanies] = React.useState<Company[]>([]);
	const [plans, setPlans] = React.useState<Plan[]>([]);
	const [selectedCompany, setSelectedCompany] = React.useState("");
	const [selectedPlan, setSelectedPlan] = React.useState("");

	React.useEffect(() => {
		getCompanies().then((res) => {
			if (res.success && res.data) setCompanies(res.data as Company[]);
		});
		getPlans().then((res) => {
			if (res.success && res.data) setPlans(res.data as Plan[]);
		});
	}, []);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!selectedCompany || !selectedPlan) return;
		setLoading(true);
		const res = await createSubscription({
			company_id: Number(selectedCompany),
			plan_code: selectedPlan,
			status: "active",
			expires_at: new Date(values.expires_at).toISOString(),
		});
		setLoading(false);
		if (res.success) navigate("/subscriptions");
	}

	return (
		<DashboardLayout>
			<div className="max-w-2xl mx-auto">
				<Form<typeof formSchema>
					schema={formSchema}
					onSubmit={onSubmit}
					defaultValues={{ expires_at: "" }}
				>
					{(field) => (
						<Card>
							<CardHeader>
								<CardTitle>
									{textGet("backoffice.subscriptions.create.title")}
								</CardTitle>
								<CardDescription>
									{textGet("backoffice.subscriptions.create.description")}
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-4">
								<div className="space-y-2">
									<Label>
										{textGet("backoffice.subscriptions.col.company")}
									</Label>
									<Select
										value={selectedCompany}
										onValueChange={(v) => v && setSelectedCompany(v)}
									>
										<SelectTrigger>
											<SelectValue
												placeholder={textGet(
													"backoffice.subscriptions.select.company",
												)}
											/>
										</SelectTrigger>
										<SelectContent>
											{companies.map((c) => (
												<SelectItem key={c.ID} value={String(c.ID)}>
													{c.trade_name}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
								</div>
								<div className="space-y-2">
									<Label>{textGet("backoffice.subscriptions.col.plan")}</Label>
									<Select
										value={selectedPlan}
										onValueChange={(v) => v && setSelectedPlan(v)}
									>
										<SelectTrigger>
											<SelectValue
												placeholder={textGet(
													"backoffice.subscriptions.select.plan",
												)}
											/>
										</SelectTrigger>
										<SelectContent>
											{plans.map((p) => (
												<SelectItem key={p.ID} value={p.code}>
													{p.name} — ${p.price.toFixed(2)}
												</SelectItem>
											))}
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
								<Button
									type="submit"
									disabled={loading || !selectedCompany || !selectedPlan}
								>
									{loading && <Spinner />}
									{textGet("backoffice.subscriptions.create")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</DashboardLayout>
	);
};

export default CreateSubscription;
