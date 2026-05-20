import React from "react";
import { useNavigate, useParams } from "react-router";
import z from "zod";
import { type Feature, getFeatures } from "@/api/feature-service";
import { getPlanByID, updatePlan } from "@/api/plan-service";
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
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const TIERS = [1, 2, 3] as const;

import {
	PLAN_LIMIT_KEYS,
	type PlanLimits,
	PlanLimitsEditor,
} from "./plan-limits-editor";
import {
	arrayToPricingsState,
	PlanPricingsEditor,
	type PricingsState,
	pricingsStateToArray,
} from "./plan-pricings-editor";

const formSchema = z.object({
	name: z.string().min(2),
});

const EditPlan = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const [loading, setLoading] = React.useState(false);
	const [initialLoading, setInitialLoading] = React.useState(true);
	const [code, setCode] = React.useState("");
	const [defaultValues, setDefaultValues] = React.useState({
		name: "",
	});
	const [features, setFeatures] = React.useState<Feature[]>([]);
	const [selectedFeatures, setSelectedFeatures] = React.useState<string[]>([]);
	const [limits, setLimits] = React.useState<PlanLimits>({
		max_users: -1,
		max_patients: -1,
		max_offices: -1,
	});
	const [tier, setTier] = React.useState<1 | 2 | 3>(1);
	const [pricings, setPricings] = React.useState<PricingsState>({});

	React.useEffect(() => {
		getFeatures().then((res) => {
			if (res.success && res.data) setFeatures(res.data as Feature[]);
		});
	}, []);

	React.useEffect(() => {
		if (!id) return;
		getPlanByID(id).then((res) => {
			if (res.success && res.data) {
				setDefaultValues({ name: res.data.name });
				setCode(res.data.code);
				const loadedTier = res.data.tier ?? 1;
				setTier(
					(loadedTier >= 1 && loadedTier <= 3 ? loadedTier : 1) as 1 | 2 | 3,
				);
				setSelectedFeatures(res.data.Features?.map((f) => f.code) ?? []);

				const props = res.data.Properties ?? {};
				const loaded: PlanLimits = {};
				for (const key of PLAN_LIMIT_KEYS) {
					const val = props[key];
					loaded[key] = val === undefined || val === null ? -1 : Number(val);
				}
				setLimits(loaded);

				const existingPricings = res.data.pricings ?? [];
				if (existingPricings.length === 0 && res.data.price > 0) {
					setPricings(
						arrayToPricingsState([{ months: 1, price: res.data.price }]),
					);
				} else {
					setPricings(arrayToPricingsState(existingPricings));
				}
			}
			setInitialLoading(false);
		});
	}, [id]);

	const toggleFeature = (featureCode: string) => {
		setSelectedFeatures((prev) =>
			prev.includes(featureCode)
				? prev.filter((f) => f !== featureCode)
				: [...prev, featureCode],
		);
	};

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!id) return;
		setLoading(true);
		const res = await updatePlan(id, {
			name: values.name,
			tier,
			feature_codes: selectedFeatures,
			properties: { ...limits } as Record<string, unknown>,
			pricings: pricingsStateToArray(pricings),
		});
		setLoading(false);
		if (res.success) navigate("/plans");
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
								<CardTitle>{textGet("backoffice.plans.edit.title")}</CardTitle>
								<CardDescription>
									{textGet("backoffice.plans.edit.description")}
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-4">
								<div>
									<span className="text-sm font-medium">
										{textGet("backoffice.plans.col.code")}
									</span>
									<p className="mt-1 text-sm font-mono text-muted-foreground bg-muted px-3 py-2 rounded-md">
										{code}
									</p>
								</div>
								<FormInput
									field={field}
									name="name"
									type="text"
									label={textGet("backoffice.plans.col.name")}
									placeholder={textGet("backoffice.plans.col.name.placeholder")}
								/>

								<div className="space-y-2">
									<Label>{textGet("backoffice.plans.col.tier")}</Label>
									<div className="flex gap-2">
										{TIERS.map((t) => (
											<button
												key={t}
												type="button"
												onClick={() => setTier(t)}
												className={cn(
													"w-12 h-10 rounded-md border text-sm font-semibold transition-colors",
													tier === t
														? "bg-primary text-primary-foreground border-primary"
														: "bg-background text-muted-foreground border-border hover:bg-muted",
												)}
											>
												{t}
											</button>
										))}
									</div>
									<p className="text-xs text-muted-foreground">
										{textGet("backoffice.plans.tier.hint")}
									</p>
								</div>

								<PlanLimitsEditor limits={limits} onChange={setLimits} />

								<PlanPricingsEditor
									pricings={pricings}
									onChange={setPricings}
								/>

								{features.length > 0 && (
									<div className="space-y-3 border-t pt-4">
										<Label>{textGet("backoffice.plans.col.features")}</Label>
										<div className="grid grid-cols-1 gap-2 max-h-48 overflow-y-auto border rounded-md p-3">
											{features.map((f) => (
												<span
													key={f.ID}
													className="flex items-center gap-2 cursor-pointer hover:bg-muted/50 rounded px-2 py-1.5 transition-colors"
												>
													<Checkbox
														checked={selectedFeatures.includes(f.code)}
														onCheckedChange={() => toggleFeature(f.code)}
													/>
													<div className="flex flex-col">
														<span className="text-sm font-medium">
															{f.name}
														</span>
														<span className="text-xs text-muted-foreground font-mono">
															{f.code}
														</span>
													</div>
												</span>
											))}
										</div>
										<p className="text-xs text-muted-foreground">
											{selectedFeatures.length}{" "}
											{textGet("backoffice.linking.selected")}
										</p>
									</div>
								)}
							</CardContent>
							<CardFooter className="flex justify-between">
								<Button
									type="button"
									variant="outline"
									onClick={() => navigate("/plans")}
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

export default EditPlan;
