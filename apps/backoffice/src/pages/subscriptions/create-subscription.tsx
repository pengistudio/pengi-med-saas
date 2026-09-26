import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	Label,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
	Spinner,
} from "@pengi/ui";
import React from "react";
import { useNavigate } from "react-router";
import {
	type Company,
	companies as companyResource,
} from "@/api/company-service";
import { type Plan, plans as planResource } from "@/api/plan-service";
import { subscriptions } from "@/api/subscription-service";
import { useText } from "@/hooks/use-text";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";
import {
	type PlanTerm,
	PlanTermFields,
} from "@/lib/subscription/plan-term-fields";

const CreateSubscription = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { saving, save } = useResourceItem(subscriptions);
	const [companies, setCompanies] = React.useState<Company[]>([]);
	const [plans, setPlans] = React.useState<Plan[]>([]);
	const [selectedCompany, setSelectedCompany] = React.useState("");
	const [term, setTerm] = React.useState<PlanTerm>({
		planCode: "",
		expiresAt: "",
	});

	React.useEffect(() => {
		companyResource.list().then((res) => {
			if (res.success && res.data) setCompanies(res.data);
		});
		planResource.list().then((res) => {
			if (res.success && res.data) setPlans(res.data);
		});
	}, []);

	async function onSubmit(e: React.FormEvent) {
		e.preventDefault();
		if (!selectedCompany || !term.planCode || !term.expiresAt) return;
		await save({
			company_id: Number(selectedCompany),
			plan_code: term.planCode,
			status: "active",
			expires_at: term.expiresAt,
		});
	}

	return (
		<ResourceEditPage>
			<div className="max-w-2xl mx-auto">
				<form onSubmit={onSubmit}>
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
								<Label>{textGet("backoffice.subscriptions.col.company")}</Label>
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
							<PlanTermFields plans={plans} value={term} onChange={setTerm} />
						</CardContent>
						<CardFooter className="flex justify-between">
							<Button
								type="button"
								variant="outline"
								onClick={() => navigate("/subscriptions")}
							>
								{textGet("backoffice.common.cancel")}
							</Button>
							<Button
								type="submit"
								disabled={
									saving ||
									!selectedCompany ||
									!term.planCode ||
									!term.expiresAt
								}
							>
								{saving && <Spinner />}
								{textGet("backoffice.subscriptions.create")}
							</Button>
						</CardFooter>
					</Card>
				</form>
			</div>
		</ResourceEditPage>
	);
};

export default CreateSubscription;
