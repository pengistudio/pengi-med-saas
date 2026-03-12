import { Loader2, Percent, Save } from "lucide-react";
import React from "react";
import { type UseFormReturn, useFormContext } from "react-hook-form";
import { z } from "zod";
import type { CatalogItem } from "@/api/billing-service";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { FormSelect } from "@/components/forms/form-select";
import { FormTextArea } from "@/components/forms/form-textarea";
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
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";
import {
	IVA_PERCENTAGE_CODES,
	IVA_PERCENTAGE_CODES_AS_NUMBER,
	TAX_CODES,
	type TaxPercentageCode,
} from "@/lib/constants";
import { ICE_CODES } from "@/lib/ice_data";

const requiredString = z
	.string()
	.min(1, "billing.catalog-item.form.error.required");
const positiveNumber = z.coerce
	.number()
	.min(0, "billing.catalog-item.form.error.positive");

// El formulario se utilizará tanto para crear como para editar
const formSchema = z.object({
	name: requiredString,
	sku: requiredString,
	description: z.string().optional(),
	unit_price: positiveNumber,
	tax: z.coerce.number().optional(),
	tax_code: z.string().optional(),
	tax_percentage_code: z.string().optional(),
	ice_tax: z.coerce.number().optional(),
	ice_tax_code: z.string().optional(),
	ice_tax_percentage_code: z.string().optional(),
});

export type FormValues = z.infer<typeof formSchema>;

const TaxAutoCalculator = () => {
	const form = useFormContext();
	const taxCode = form.watch("tax_code");
	const taxPercentageCode = form.watch("tax_percentage_code");
	const iceCode = form.watch("ice_tax_code");
	const icePercentageCode = form.watch("ice_tax_percentage_code");

	React.useEffect(() => {
		let rate = 0;
		let iceRate = 0;
		if (taxCode === "2" || taxCode === "3") {
			rate =
				IVA_PERCENTAGE_CODES_AS_NUMBER[
					taxPercentageCode as TaxPercentageCode
				] || 0;
		}
		if (iceCode === "3") {
			// ICE
			const iceItem = ICE_CODES.find(
				(item) => item.code.toString() === icePercentageCode,
			);
			if (iceItem) {
				iceRate = iceItem.rate;
			}
		}
		const currentTax = form.getValues("tax");
		const currentIceTax = form.getValues("ice_tax");

		if (rate !== currentTax) {
			form.setValue("tax", rate);
		}
		if (iceRate !== currentIceTax) {
			form.setValue("ice_tax", iceRate);
		}
	}, [taxCode, taxPercentageCode, iceCode, icePercentageCode, form]);

	return null;
};

export interface CatalogItemFormProps {
	initialData?: CatalogItem; // Si viene, es edición
	onSubmit: (values: FormValues) => Promise<void>;
	loading?: boolean;
}

const CatalogItemForm = ({
	initialData,
	onSubmit,
	loading = false,
}: CatalogItemFormProps) => {
	const { textGet } = useText();

	const isEditing = !!initialData;

	const defaultValues: FormValues = {
		name: initialData?.name || "",
		sku: initialData?.sku || "",
		description: initialData?.description || "",
		unit_price: initialData?.unit_price || 0,
		tax: initialData?.tax || 0,
		tax_code: initialData?.tax_code || "2",
		tax_percentage_code: initialData?.tax_percentage_code || "4",
		ice_tax: initialData?.ice_tax || 0,
		ice_tax_code: initialData?.ice_tax_code || "3",
		ice_tax_percentage_code: initialData?.ice_tax_percentage_code || "3000",
	};

	const [isICE, setIsICE] = React.useState(
		Boolean(initialData?.ice_tax && initialData?.ice_tax > 0),
	);

	return (
		<Form schema={formSchema} onSubmit={onSubmit} defaultValues={defaultValues}>
			{(field) => (
				<Card className="max-w-4xl mx-auto">
					<TaxAutoCalculator />
					<CardHeader>
						<CardTitle>
							{textGet(
								isEditing
									? "billing.catalog-item.form.edit.title"
									: "billing.catalog-item.form.create.title",
							)}
						</CardTitle>
						<CardDescription>
							{textGet(
								isEditing
									? "billing.catalog-item.form.edit.description"
									: "billing.catalog-item.form.create.description",
							)}
						</CardDescription>
					</CardHeader>
					<CardContent className="space-y-4">
						<div className="grid md:grid-cols-2 grid-cols-1 gap-2 md:gap-4">
							<FormInput
								field={field}
								name="name"
								placeholder={textGet("billing.catalog-item.form.name")}
								label={textGet("billing.catalog-item.form.name")}
							/>
							<FormInput
								field={field}
								name="sku"
								placeholder={textGet("billing.catalog-item.form.sku")}
								label={textGet("billing.catalog-item.form.sku")}
							/>
						</div>

						<FormTextArea
							field={field}
							name="description"
							placeholder={textGet("billing.catalog-item.form.description")}
							label={textGet("billing.catalog-item.form.description")}
							isOptional
						/>

						<div className="grid md:grid-cols-4 grid-cols-1 gap-2 md:gap-4">
							<FormInput
								field={field}
								name="unit_price"
								type="number"
								placeholder="0.00"
								label={textGet("billing.catalog-item.form.unit_price")}
								step="0.01"
							/>
							<FormSelect
								field={field}
								name="tax_code"
								placeholder={textGet(
									"billing.catalog-item.form.tax.code.placeholder",
								)}
								label={textGet("billing.catalog-item.form.tax.code")}
								options={handleTaxCodeOptions()}
							/>
							<FormSelect
								field={field}
								name="tax_percentage_code"
								placeholder={textGet(
									"billing.catalog-item.form.tax.percentage.placeholder",
								)}
								label={textGet("billing.catalog-item.form.tax.percentage")}
								options={handleTaxPercentageOptions()}
							/>
							<FormInput
								field={field}
								name="tax"
								type="number"
								placeholder={textGet("billing.catalog-item.form.tax")}
								label={textGet("billing.catalog-item.form.tax")}
								autoComplete="tax"
								startAddon={<Percent className="w-4 h-4 text-green-600" />}
								step="0.01"
								disabled
								isOptional
							/>
						</div>

						<div className="flex items-center gap-3 py-2">
							<Checkbox
								id="toggleICE"
								checked={isICE}
								onCheckedChange={(checked: boolean) =>
									handleICECheckedChange(
										checked,
										field as unknown as UseFormReturn<FormValues>,
									)
								}
								className="hover:cursor-pointer"
							/>
							<Label
								htmlFor="toggleICE"
								className="hover:cursor-pointer font-bold text-base"
							>
								<Text uuid="billing.catalog-item.form.ice.apply" type="span" />
							</Label>
						</div>

						<div className="grid md:grid-cols-3 gap-2">
							<FormSelect
								field={field}
								name="ice_tax_code"
								placeholder={textGet(
									"billing.catalog-item.form.ice.tax.code.placeholder",
								)}
								label={textGet("billing.catalog-item.form.ice.tax.code")}
								options={handleIceTaxCodeOptions()}
								disabled={!isICE}
							/>
							<FormSelect
								field={field}
								name="ice_tax_percentage_code"
								placeholder={textGet(
									"billing.catalog-item.form.ice.tax.percentage.placeholder",
								)}
								label={textGet("billing.catalog-item.form.ice.tax.percentage")}
								options={handleIceTaxPercentageOptions()}
								disabled={!isICE}
							/>
							<FormInput
								field={field}
								name="ice_tax"
								type="number"
								placeholder={textGet(
									"billing.catalog-item.form.ice.tax.placeholder",
								)}
								label={textGet("billing.catalog-item.form.ice_tax")}
								startAddon={<Percent className="w-4 h-4 text-green-600" />}
								step="0.01"
								disabled
								isOptional
							/>
						</div>
					</CardContent>
					<CardFooter>
						<Button type="submit" disabled={loading}>
							{loading ? (
								<Loader2 className="mr-2 h-4 w-4 animate-spin" />
							) : (
								<Save className="mr-2 h-4 w-4" />
							)}
							{textGet(
								isEditing
									? "billing.catalog-item.form.submit.edit"
									: "billing.catalog-item.form.submit.create",
							)}
						</Button>
					</CardFooter>
				</Card>
			)}
		</Form>
	);

	function handleTaxCodeOptions() {
		return Object.entries(TAX_CODES)
			.filter(([key]) => key !== "3")
			.map(([key, value]) => ({
				value: key,
				label: value,
			}));
	}

	function handleIceTaxCodeOptions() {
		return Object.entries(TAX_CODES)
			.filter(([key]) => key === "3")
			.map(([key, value]) => ({
				value: key,
				label: value,
			}));
	}

	function handleIceTaxPercentageOptions() {
		return ICE_CODES.map((ice) => ({
			value: ice.code.toString(),
			label: ice.description,
		}));
	}

	function handleTaxPercentageOptions() {
		// IVA percentage codes
		return Object.entries(IVA_PERCENTAGE_CODES).map(([key, value]) => ({
			value: key,
			label: value,
		}));
	}

	function handleICECheckedChange(
		isChecked: boolean,
		formMethods: UseFormReturn<FormValues>,
	) {
		setIsICE(isChecked);
		formMethods.setValue("ice_tax_code", "3");
		formMethods.setValue("ice_tax_percentage_code", "3000");
		formMethods.setValue("ice_tax", 0);
	}
};

export default CatalogItemForm;
