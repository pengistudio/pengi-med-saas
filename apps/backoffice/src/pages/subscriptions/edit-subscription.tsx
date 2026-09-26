import { useText } from "@pengi/shared";
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
import { useNavigate, useParams } from "react-router";
import { type Plan, plans as planResource } from "@/api/plan-service";
import { subscriptions } from "@/api/subscription-service";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";
import {
	type PlanTerm,
	PlanTermFields,
} from "@/lib/subscription/plan-term-fields";
import { expiryDate } from "@/lib/subscription/term";

const STATUSES = ["active", "expired", "cancelled"] as const;

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
	const [term, setTerm] = React.useState<PlanTerm>({
		planCode: "",
		expiresAt: "",
	});
	const [status, setStatus] = React.useState("active");

	React.useEffect(() => {
		planResource.list().then((res) => {
			if (res.success && res.data) setPlans(res.data);
		});
	}, []);

	React.useEffect(() => {
		if (!subscription) return;
		setTerm({
			planCode: subscription.plan_code,
			expiresAt: expiryDate(subscription.expires_at),
		});
		setStatus(subscription.status);
	}, [subscription]);

	async function onSubmit(e: React.FormEvent) {
		e.preventDefault();
		if (!term.planCode || !term.expiresAt) return;
		await save({
			plan_code: term.planCode,
			status,
			expires_at: term.expiresAt,
		});
	}

	return (
		<ResourceEditPage loading={loading}>
			<div className="max-w-2xl mx-auto">
				<form onSubmit={onSubmit}>
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
							<PlanTermFields
								plans={plans}
								value={term}
								onChange={setTerm}
								suggestOnPlanChange={false}
							/>
							<div className="space-y-2">
								<Label>{textGet("backoffice.subscriptions.col.status")}</Label>
								<Select value={status} onValueChange={(v) => v && setStatus(v)}>
									<SelectTrigger>
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										{STATUSES.map((s) => (
											<SelectItem key={s} value={s}>
												{textGet(`subscription.status.${s}`)}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
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
				</form>
			</div>
		</ResourceEditPage>
	);
};

export default EditSubscription;
