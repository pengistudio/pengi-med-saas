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
import { useNavigate } from "react-router";
import z from "zod";
import {
	type Feature,
	features as featureResource,
} from "@/api/feature-service";
import { plans } from "@/api/plan-service";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";
import { FormSection, type Tier, TierSelect } from "./form-section";
import { type PlanLimits, PlanLimitsEditor } from "./plan-limits-editor";
import {
	PlanPricingsEditor,
	type PricingsState,
	pricingsStateToArray,
} from "./plan-pricings-editor";

const formSchema = z.object({
	name: z.string().min(2),
	code: z.string().min(2),
});

const CreatePlan = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { saving, save } = useResourceItem(plans);
	const [features, setFeatures] = React.useState<Feature[]>([]);
	const [selectedFeatures, setSelectedFeatures] = React.useState<string[]>([]);
	const [limits, setLimits] = React.useState<PlanLimits>({
		max_users: -1,
		max_patients: -1,
		max_offices: -1,
		// Pengi pays every WhatsApp template: a new plan starts capped, not
		// unlimited (same default as the migration of existing plans).
		max_whatsapp_messages: 300,
	});
	const [tier, setTier] = React.useState<Tier>(1);
	const [storageQuotaMb, setStorageQuotaMb] = React.useState(0);
	const [pricings, setPricings] = React.useState<PricingsState>({});

	React.useEffect(() => {
		featureResource.list().then((res) => {
			if (res.success && res.data) setFeatures(res.data as Feature[]);
		});
	}, []);

	const toggleFeature = (code: string) => {
		setSelectedFeatures((prev) =>
			prev.includes(code) ? prev.filter((f) => f !== code) : [...prev, code],
		);
	};

	async function onSubmit(values: z.infer<typeof formSchema>) {
		await save({
			name: values.name,
			code: values.code,
			tier,
			price: 0,
			feature_codes: selectedFeatures,
			properties: { ...limits } as Record<string, unknown>,
			storage_quota_mb: storageQuotaMb,
			pricings: pricingsStateToArray(pricings),
		});
	}

	return (
		<ResourceEditPage>
			<div className="max-w-2xl mx-auto">
				<Form<typeof formSchema>
					schema={formSchema}
					onSubmit={onSubmit}
					defaultValues={{ name: "", code: "" }}
				>
					{(field) => (
						<Card>
							<CardHeader>
								<CardTitle>
									{textGet("backoffice.plans.create.title")}
								</CardTitle>
								<CardDescription>
									{textGet("backoffice.plans.create.description")}
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
									<FormInput
										field={field}
										name="code"
										type="text"
										label={textGet("backoffice.plans.col.code")}
										placeholder="ENT, PRO, BASIC..."
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
									{textGet("backoffice.plans.create")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</ResourceEditPage>
	);
};

export default CreatePlan;
