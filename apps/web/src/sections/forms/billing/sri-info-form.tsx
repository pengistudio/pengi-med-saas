import { Button, Checkbox, FormInput, Label, Text } from "@pengi/ui";
import { Loader2, Save } from "lucide-react";
import { useState } from "react";
import * as z from "zod";
import { type SriStatus, updateSriInfo } from "@/api/tenant-service";
import { Form } from "@/components/forms/form";
import { useText } from "@/hooks/use-text";

export const sriInfoSchema = z.object({
	tax_id: z.string().min(1, "form.validation.required"),
	trade_name: z.string().min(1, "form.validation.required"),
	corporate_name: z.string().min(1, "form.validation.required"),
	address: z.string().min(1, "form.validation.required"),
	accounting_obliged: z.boolean().default(false),
	special_contributor_number: z.string().default(""),
	microenterprise_regime: z.boolean().default(false),
	withholding_agent: z.string().default(""),
	rimpe_taxpayer: z.string().default(""),
});

type FormValues = z.infer<typeof sriInfoSchema>;

interface SriInfoFormProps {
	initialData?: SriStatus | null;
	onSuccess?: () => void;
}

export function SriInfoForm({ initialData, onSuccess }: SriInfoFormProps) {
	const [loading, setLoading] = useState(false);
	const { textGet } = useText();

	const defaultValues: FormValues = {
		tax_id: initialData?.tax_id || "",
		trade_name: initialData?.trade_name || "",
		corporate_name: initialData?.corporate_name || "",
		address: initialData?.address || "",
		accounting_obliged: initialData?.accounting_obliged || false,
		special_contributor_number: initialData?.special_contributor_number || "",
		microenterprise_regime: initialData?.microenterprise_regime || false,
		withholding_agent: initialData?.withholding_agent || "",
		rimpe_taxpayer: initialData?.rimpe_taxpayer || "",
	};

	const onSubmit = async (values: FormValues) => {
		setLoading(true);
		try {
			const response = await updateSriInfo(values);
			if (response.success && onSuccess) {
				onSuccess();
			}
		} finally {
			setLoading(false);
		}
	};

	return (
		<Form
			schema={sriInfoSchema}
			defaultValues={defaultValues}
			onSubmit={onSubmit}
		>
			{(field) => (
				<div className="space-y-6">
					<div className="grid gap-4 md:grid-cols-2">
						<FormInput
							field={field}
							name="tax_id"
							label={textGet("billing.sri.info.tax_id.label")}
							placeholder="0999999999001"
						/>
						<FormInput
							field={field}
							name="corporate_name"
							label={textGet("billing.sri.info.corporate_name.label")}
							placeholder="Mi Empresa S.A."
						/>
						<FormInput
							field={field}
							name="trade_name"
							label={textGet("billing.sri.info.trade_name.label")}
							placeholder="Mi Tienda"
						/>
						<FormInput
							field={field}
							name="address"
							label={textGet("billing.sri.info.address.label")}
							placeholder="Av. Principal y Secundaria"
						/>
					</div>
					<div className="flex items-center space-x-2">
						<Checkbox
							id="accounting_obliged"
							checked={field.watch("accounting_obliged")}
							onCheckedChange={(checked) =>
								field.setValue("accounting_obliged", checked as boolean)
							}
						/>
						<Label
							htmlFor="accounting_obliged"
							className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
						>
							<Text uuid="billing.sri.info.accounting_obliged.label" />
						</Label>
					</div>

					<div className="space-y-4 border-t pt-4">
						<div>
							<span className="block text-sm font-medium">
								<Text uuid="billing.sri.info.tax_classification.title" />
							</span>
							<span className="text-muted-foreground block text-xs">
								<Text uuid="billing.sri.info.tax_classification.desc" />
							</span>
						</div>
						<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
							<FormInput
								field={field}
								name="special_contributor_number"
								label={textGet(
									"billing.sri.info.special_contributor_number.label",
								)}
								placeholder="0000"
							/>
							<FormInput
								field={field}
								name="withholding_agent"
								label={textGet("billing.sri.info.withholding_agent.label")}
								placeholder="0000"
							/>
							<FormInput
								field={field}
								name="rimpe_taxpayer"
								label={textGet("billing.sri.info.rimpe_taxpayer.label")}
								placeholder="CONTRIBUYENTE RÉGIMEN RIMPE"
							/>
						</div>
						<div className="flex items-center space-x-2">
							<Checkbox
								id="microenterprise_regime"
								checked={field.watch("microenterprise_regime")}
								onCheckedChange={(checked) =>
									field.setValue("microenterprise_regime", checked as boolean)
								}
							/>
							<Label
								htmlFor="microenterprise_regime"
								className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
							>
								<Text uuid="billing.sri.info.microenterprise_regime.label" />
							</Label>
						</div>
					</div>

					<Button type="submit" disabled={loading} className="w-fit">
						{loading ? (
							<Loader2 className="mr-2 h-4 w-4 animate-spin" />
						) : (
							<Save className="mr-2 h-4 w-4" />
						)}
						<Text uuid="form.save" />
					</Button>
				</div>
			)}
		</Form>
	);
}
