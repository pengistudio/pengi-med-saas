import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import { type Feature, getFeatures } from "@/api/feature-service";
import { createPlan } from "@/api/plan-service";
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
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { type PlanLimits, PlanLimitsEditor } from "./plan-limits-editor";

const formSchema = z.object({
	name: z.string().min(2),
	code: z.string().min(2),
	price: z.coerce.number().min(0),
});

const CreatePlan = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [loading, setLoading] = React.useState(false);
	const [features, setFeatures] = React.useState<Feature[]>([]);
	const [selectedFeatures, setSelectedFeatures] = React.useState<string[]>([]);
	const [limits, setLimits] = React.useState<PlanLimits>({
		max_users: -1,
		max_patients: -1,
		max_offices: -1,
	});

	React.useEffect(() => {
		getFeatures().then((res) => {
			if (res.success && res.data) setFeatures(res.data as Feature[]);
		});
	}, []);

	const toggleFeature = (code: string) => {
		setSelectedFeatures((prev) =>
			prev.includes(code) ? prev.filter((f) => f !== code) : [...prev, code],
		);
	};

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoading(true);
		const res = await createPlan({
			...values,
			feature_codes: selectedFeatures,
			properties: limits as Record<string, unknown>,
		});
		setLoading(false);
		if (res.success) navigate("/plans");
	}

	return (
		<DashboardLayout>
			<div className="max-w-2xl mx-auto">
				<Form<typeof formSchema>
					schema={formSchema}
					onSubmit={onSubmit}
					defaultValues={{ name: "", code: "", price: 0 }}
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
							<CardContent className="space-y-4">
								<FormInput
									field={field}
									name="name"
									type="text"
									label={textGet("backoffice.plans.col.name")}
									placeholder={textGet("backoffice.plans.col.name.placeholder")}
								/>
								<FormInput
									field={field}
									name="code"
									type="text"
									label={textGet("backoffice.plans.col.code")}
									placeholder="ENT, PRO, BASIC..."
								/>
								<FormInput
									field={field}
									name="price"
									type="number"
									label={textGet("backoffice.plans.col.price")}
									placeholder="99.99"
								/>

								<PlanLimitsEditor limits={limits} onChange={setLimits} />

								{features.length > 0 && (
									<div className="space-y-3">
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
									{textGet("backoffice.plans.create")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</DashboardLayout>
	);
};

export default CreatePlan;
