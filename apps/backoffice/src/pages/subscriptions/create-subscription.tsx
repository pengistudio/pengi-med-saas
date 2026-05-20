import React from "react";
import { useNavigate } from "react-router";
import { type Company, getCompanies } from "@/api/company-service";
import { getPlans, type Plan } from "@/api/plan-service";
import { createSubscription } from "@/api/subscription-service";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
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

function addMonths(date: Date, months: number): Date {
	const result = new Date(date);
	result.setMonth(result.getMonth() + months);
	return result;
}

function toDateInputValue(date: Date): string {
	return date.toISOString().split("T")[0];
}

const CreateSubscription = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [loading, setLoading] = React.useState(false);
	const [companies, setCompanies] = React.useState<Company[]>([]);
	const [plans, setPlans] = React.useState<Plan[]>([]);
	const [selectedCompany, setSelectedCompany] = React.useState("");
	const [selectedPlan, setSelectedPlan] = React.useState("");
	const [selectedMonths, setSelectedMonths] = React.useState<number>(1);
	const [expiresAt, setExpiresAt] = React.useState("");

	React.useEffect(() => {
		getCompanies().then((res) => {
			if (res.success && res.data) setCompanies(res.data as Company[]);
		});
		getPlans().then((res) => {
			if (res.success && res.data) setPlans(res.data as Plan[]);
		});
	}, []);

	const currentPlan = plans.find((p) => p.code === selectedPlan);
	const sortedPricings = React.useMemo(() => {
		if (!currentPlan?.pricings?.length) return [];
		return [...currentPlan.pricings].sort((a, b) => a.months - b.months);
	}, [currentPlan]);

	const selectedPricing = sortedPricings.find(
		(p) => p.months === selectedMonths,
	);

	// Reset months & auto-calculate expiration when plan changes
	React.useEffect(() => {
		if (sortedPricings.length > 0) {
			const firstMonths = sortedPricings[0].months;
			setSelectedMonths(firstMonths);
			setExpiresAt(toDateInputValue(addMonths(new Date(), firstMonths)));
		} else {
			setSelectedMonths(1);
			setExpiresAt(toDateInputValue(addMonths(new Date(), 1)));
		}
	}, [sortedPricings]);

	const handleSelectMonths = (months: number) => {
		setSelectedMonths(months);
		setExpiresAt(toDateInputValue(addMonths(new Date(), months)));
	};

	async function onSubmit(e: React.FormEvent) {
		e.preventDefault();
		if (!selectedCompany || !selectedPlan || !expiresAt) return;
		setLoading(true);
		const res = await createSubscription({
			company_id: Number(selectedCompany),
			plan_code: selectedPlan,
			status: "active",
			expires_at: new Date(expiresAt).toISOString(),
		});
		setLoading(false);
		if (res.success) navigate("/subscriptions");
	}

	return (
		<DashboardLayout>
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
												{p.name}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>

							{/* Period selector — shown when the selected plan has pricings */}
							{sortedPricings.length > 0 && (
								<div className="space-y-2">
									<Label>{textGet("backoffice.plans.pricings.title")}</Label>
									<div className="flex flex-wrap gap-2">
										{sortedPricings.map((p) => (
											<button
												key={p.months}
												type="button"
												onClick={() => handleSelectMonths(p.months)}
												className={cn(
													"px-3 py-1.5 rounded-md text-sm font-medium transition-colors border",
													selectedMonths === p.months
														? "bg-primary text-primary-foreground border-primary"
														: "bg-muted text-muted-foreground border-transparent hover:bg-muted/80",
												)}
											>
												{textGet(`subscription.plans.period.${p.months}`)} — $
												{p.price.toFixed(2)}
											</button>
										))}
									</div>
									{selectedPricing && selectedPricing.months > 1 && (
										<p className="text-xs text-muted-foreground">
											≈ $
											{(selectedPricing.price / selectedPricing.months).toFixed(
												2,
											)}{" "}
											/ mes
										</p>
									)}
								</div>
							)}

							<div className="space-y-2">
								<Label>{textGet("backoffice.subscriptions.col.expires")}</Label>
								<Input
									type="date"
									value={expiresAt}
									onChange={(e) => setExpiresAt(e.target.value)}
									required
								/>
							</div>
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
				</form>
			</div>
		</DashboardLayout>
	);
};

export default CreateSubscription;
