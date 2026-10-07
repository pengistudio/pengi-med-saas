import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	Checkbox,
	Form,
	FormInput,
	Spinner,
} from "@pengi/ui";
import React from "react";
import { useNavigate, useParams } from "react-router";
import z from "zod";
import {
	type Feature,
	features as featureResource,
} from "@/api/feature-service";
import { plans } from "@/api/plan-service";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";

import { FormSection, type Tier, TierSelect } from "./form-section";
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
	const { item: plan, loading, saving, save } = useResourceItem(plans, id);
	const defaultValues = { name: plan?.name ?? "" };
	const code = plan?.code ?? "";
	const [features, setFeatures] = React.useState<Feature[]>([]);
	const [selectedFeatures, setSelectedFeatures] = React.useState<string[]>([]);
	const [limits, setLimits] = React.useState<PlanLimits>({
		max_users: -1,
		max_patients: -1,
		max_offices: -1,
		max_whatsapp_messages: -1,
	});
	const [tier, setTier] = React.useState<Tier>(1);
	const [storageQuotaMb, setStorageQuotaMb] = React.useState(0);
	const [pricings, setPricings] = React.useState<PricingsState>({});

	React.useEffect(() => {
		featureResource.list().then((res) => {
			if (res.success && res.data) setFeatures(res.data as Feature[]);
		});
	}, []);

	// Load the saved values once the plan arrives (adjust state during render).
	const [prevPlan, setPrevPlan] = React.useState<typeof plan>();
	if (plan !== prevPlan) {
		setPrevPlan(plan);
		if (plan) {
			const loadedTier = plan.tier ?? 1;
			setTier((loadedTier >= 1 && loadedTier <= 3 ? loadedTier : 1) as Tier);
			setSelectedFeatures(plan.Features?.map((f) => f.code) ?? []);

			const props = plan.Properties ?? {};
			const loaded: PlanLimits = {};
			for (const key of PLAN_LIMIT_KEYS) {
				const val = props[key];
				loaded[key] = val === undefined || val === null ? -1 : Number(val);
			}
			setLimits(loaded);
			setStorageQuotaMb(plan.storage_quota_mb ?? 0);

			const existingPricings = plan.pricings ?? [];
			if (existingPricings.length === 0 && plan.price > 0) {
				setPricings(arrayToPricingsState([{ months: 1, price: plan.price }]));
			} else {
				setPricings(arrayToPricingsState(existingPricings));
			}
		}
	}

	const toggleFeature = (featureCode: string) => {
		setSelectedFeatures((prev) =>
			prev.includes(featureCode)
				? prev.filter((f) => f !== featureCode)
				: [...prev, featureCode],
		);
	};

	async function onSubmit(values: z.infer<typeof formSchema>) {
		await save({
			name: values.name,
			tier,
			feature_codes: selectedFeatures,
			properties: { ...limits } as Record<string, unknown>,
			storage_quota_mb: storageQuotaMb,
			pricings: pricingsStateToArray(pricings),
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
								<div className="flex items-center gap-2">
									<CardTitle>
										{textGet("backoffice.plans.edit.title")}
									</CardTitle>
									<span className="rounded-md border bg-muted px-2 py-0.5 font-mono text-xs text-muted-foreground">
										{code}
									</span>
								</div>
								<CardDescription>
									{textGet("backoffice.plans.edit.description")}
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-6">
								<FormSection
									title={textGet("backoffice.plans.section.general.title")}
									description={textGet(
										"backoffice.plans.section.general.description",
									)}
								>
									<FormInput
										field={field}
										name="name"
										type="text"
										label={textGet("backoffice.plans.col.name")}
										placeholder={textGet(
											"backoffice.plans.col.name.placeholder",
										)}
									/>
									<TierSelect value={tier} onChange={setTier} />
								</FormSection>

								<FormSection
									title={textGet("backoffice.plans.limits.title")}
									description={textGet(
										"backoffice.plans.section.limits.description",
									)}
								>
									<PlanLimitsEditor
										limits={limits}
										onChange={setLimits}
										storageQuotaMb={storageQuotaMb}
										onStorageQuotaChange={setStorageQuotaMb}
									/>
								</FormSection>

								<FormSection
									title={textGet("backoffice.plans.pricings.title")}
									description={textGet("backoffice.plans.pricings.description")}
								>
									<PlanPricingsEditor
										pricings={pricings}
										onChange={setPricings}
									/>
								</FormSection>

								{features.length > 0 && (
									<FormSection
										title={textGet("backoffice.plans.col.features")}
										description={textGet(
											"backoffice.plans.section.features.description",
										)}
									>
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
									</FormSection>
								)}
							</CardContent>
							<CardFooter className="flex justify-between">
								<Button
									type="button"
									variant="outline"
									onClick={() => navigate("/plans")}
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

export default EditPlan;
