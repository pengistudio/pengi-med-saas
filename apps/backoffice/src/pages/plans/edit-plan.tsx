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
	cn,
	Form,
	FormInput,
	Label,
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
	const { item: plan, loading, saving, save } = useResourceItem(plans, id);
	const defaultValues = { name: plan?.name ?? "" };
	const code = plan?.code ?? "";
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
			setTier(
				(loadedTier >= 1 && loadedTier <= 3 ? loadedTier : 1) as 1 | 2 | 3,
			);
			setSelectedFeatures(plan.Features?.map((f) => f.code) ?? []);

			const props = plan.Properties ?? {};
			const loaded: PlanLimits = {};
			for (const key of PLAN_LIMIT_KEYS) {
				const val = props[key];
				loaded[key] = val === undefined || val === null ? -1 : Number(val);
			}
			setLimits(loaded);

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
