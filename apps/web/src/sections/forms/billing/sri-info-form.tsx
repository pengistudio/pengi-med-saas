import { Loader2, Save } from "lucide-react";
import { useState } from "react";
import * as z from "zod";
import { type SriStatus, updateSriInfo } from "@/api/tenant-service";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";

export const sriInfoSchema = z.object({
	tax_id: z.string().min(1, "form.validation.required"),
	trade_name: z.string().min(1, "form.validation.required"),
	corporate_name: z.string().min(1, "form.validation.required"),
	address: z.string().min(1, "form.validation.required"),
	accounting_obliged: z.boolean().default(false),
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
	};

	const onSubmit = async (values: FormValues) => {
		setLoading(true);
		const response = await updateSriInfo(values);
		if (response.success && onSuccess) {
			onSuccess();
		}
		setLoading(false);
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
