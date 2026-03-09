import { Loader2, Plus, Trash } from "lucide-react";
import { useEffect, useState } from "react";
import { type UseFormReturn, useFieldArray } from "react-hook-form";
import { useNavigate } from "react-router";
import * as z from "zod";
import { createInvoice } from "@/api/billing-service";
import {
	getAllPatientsWithLastFollowUp,
	type Patient,
} from "@/api/clinical-service";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { FormSelect } from "@/components/forms/form-select";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";
import useToast from "@/hooks/use-toast";
import {
	IVA_PERCENTAGE_CODES_AS_NUMBER,
	PAYMENT_LABELS,
	type TaxPercentageCode,
} from "@/lib/constants";

const invoiceItemSchema = z.object({
	description: z.string().min(1, "validation.required"),
	quantity: z.number().min(1, "validation.min_1"),
	unit_price: z.number().min(0, "validation.min_0"),
	discount: z.number().min(0, "validation.min_0"),
	tax_rate: z.string().min(1, "validation.required"),
	ice_tax: z.number().min(0, "validation.min_0"),
});

const formSchema = z.object({
	patient_id: z.string().optional(),
	payment_method: z.string().min(1, "validation.required"),
	term: z.coerce.number().min(0, "validation.min_0"),
	time_unit: z.string().min(1, "validation.required"),
	establishment_code: z.string().min(1, "validation.required"),
	emission_point_code: z.string().min(1, "validation.required"),
	items: z.array(invoiceItemSchema).min(1, "validation.min_1_item"),
});

export function InvoiceForm() {
	const [patients, setPatients] = useState<Patient[]>([]);
	const [loading, setLoading] = useState(false);
	const { successToast, errorToast } = useToast();
	const navigate = useNavigate();

	useEffect(() => {
		getAllPatientsWithLastFollowUp().then((res) => {
			if (res.success && res.data) {
				setPatients(res.data);
			}
		});
	}, []);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoading(true);

		const itemsWatch = values.items;
		const subTotal = itemsWatch.reduce(
			(sum, item) => sum + (item.quantity * item.unit_price - item.discount),
			0,
		);
		const totalTax = itemsWatch.reduce((sum, item) => {
			const rate =
				IVA_PERCENTAGE_CODES_AS_NUMBER[item.tax_rate as TaxPercentageCode] || 0;
			return sum + (item.quantity * item.unit_price - item.discount) * rate;
		}, 0);
		const total = subTotal + totalTax;

		const payload = {
			...values,
			patient_id: values.patient_id ? parseInt(values.patient_id, 10) : 0,
			items: values.items.map((item) => ({
				...item,
				product_id: 1, // Added default since it does not exist on UI
				total:
					item.quantity * item.unit_price -
					item.discount +
					(item.quantity * item.unit_price - item.discount) *
						(IVA_PERCENTAGE_CODES_AS_NUMBER[
							item.tax_rate as TaxPercentageCode
						] || 0),
			})),
			subtotal: subTotal,
			discount: itemsWatch.reduce((sum, item) => sum + item.discount, 0),
			tax_total: totalTax,
			total: total,
		};

		const res = await createInvoice(payload);
		if (res.success) {
			successToast(res.message);
			navigate("/clinical/billing");
		} else {
			errorToast(null, res.message as string);
		}
		setLoading(false);
	}

	return (
		<Form
			schema={formSchema}
			onSubmit={onSubmit}
			defaultValues={{
				patient_id: "",
				items: [
					{
						description: "Consulta Médica",
						quantity: 1,
						unit_price: 0,
						discount: 0,
						tax_rate: "0",
						ice_tax: 0,
					},
				],
			}}
		>
			{(form) => (
				<InvoiceFormInner form={form} patients={patients} loading={loading} />
			)}
		</Form>
	);
}

function InvoiceFormInner({
	form,
	patients,
	loading,
}: {
	// biome-ignore lint/suspicious/noExplicitAny: UseFormReturn<any> is acceptable here
	form: UseFormReturn<any>;
	patients: Patient[];
	loading: boolean;
}) {
	const { fields, append, remove } = useFieldArray({
		name: "items",
		control: form.control,
	});
	const { textGet } = useText();
	const navigate = useNavigate();
	const [isFinalCustomer, setIsFinalCustomer] = useState(false);

	const itemsWatch: z.infer<typeof invoiceItemSchema>[] = form.watch("items");

	const subtotal = (itemsWatch || []).reduce(
		(sum, item) => sum + (item.quantity * item.unit_price - item.discount),
		0,
	);
	const discountTotal = (itemsWatch || []).reduce(
		(sum, item) => sum + item.discount,
		0,
	);
	const totalTax = (itemsWatch || []).reduce((sum, item) => {
		const rate =
			IVA_PERCENTAGE_CODES_AS_NUMBER[item.tax_rate as TaxPercentageCode] || 0;
		return sum + (item.quantity * item.unit_price - item.discount) * rate;
	}, 0);
	const total = subtotal + totalTax;

	const handleFinalCustomerChange = (isChecked: boolean) => {
		setIsFinalCustomer(isChecked);
		if (isChecked) {
			form.setValue("patient_id", "");
		}
	};

	function renderPaymentMethod() {
		return Object.entries(PAYMENT_LABELS).map(([value, label]) => ({
			label,
			value,
		}));
	}

	function renderTimeUnits() {
		const units = [
			{ label: textGet("common.days") || "Días", value: "dias" },
			{ label: textGet("common.months") || "Meses", value: "meses" },
			{ label: textGet("common.years") || "Años", value: "anios" },
		];
		return units;
	}

	return (
		<div className="space-y-6">
			<Card>
				<CardHeader>
					<CardTitle>
						<Text uuid="billing.invoice.establishment.title" />
					</CardTitle>
					<CardDescription>
						<Text uuid="billing.invoice.establishment.desc" />
					</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					<div className="grid lg:grid-cols-2 gap-4">
						<FormInput
							field={form}
							name="establishment_code"
							label="Código de Establecimiento"
							placeholder="ej. 001"
						/>
						<FormInput
							field={form}
							name="emission_point_code"
							label="Punto de Emisión"
							placeholder="ej. 001"
						/>
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardHeader>
					<CardTitle>
						<Text uuid="billing.invoice.client.title" />
					</CardTitle>
					<CardDescription>
						<Text uuid="billing.invoice.client.desc" />
					</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					<div className="flex flex-row items-center gap-2">
						<Switch
							checked={isFinalCustomer}
							onCheckedChange={handleFinalCustomerChange}
						/>
						<div>
							<span className="block font-medium">
								<Text uuid="billing.invoice.final_consumer" />
							</span>
							<span className="text-muted-foreground block text-sm">
								<Text uuid="billing.invoice.final_consumer.desc" />
							</span>
						</div>
					</div>

					<div className="flex flex-row gap-2 items-end w-full">
						<FormSelect
							name="patient_id"
							label={textGet("billing.invoice.patient") || "Paciente"}
							placeholder={
								textGet("billing.invoice.patient.placeholder") ||
								"Seleccionar paciente..."
							}
							field={form}
							options={patients.map((p) => ({
								label: `${p.first_name} ${p.last_name} - ${p.document}`,
								value: p.ID.toString(),
							}))}
						/>
						<Button
							type="button"
							onClick={() => navigate("/clinical/patients/create")}
						>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="billing.invoice.new_patient" />
						</Button>
					</div>

					<div className="grid md:grid-cols-3 grid-cols-1 gap-4 mt-4">
						<FormSelect
							field={form}
							name="payment_method"
							options={renderPaymentMethod()}
							label={
								textGet("billing.invoice.payment.method") || "Método de pago"
							}
							placeholder={
								textGet("billing.invoice.payment.method.placeholder") || ""
							}
						/>
						<FormSelect
							name="time_unit"
							label={textGet("billing.invoice.time.unit") || "Unidad"}
							field={form}
							options={renderTimeUnits()}
							placeholder={
								textGet("billing.invoice.time.unit.placeholder") || ""
							}
						/>
						<FormInput
							name="term"
							label={textGet("billing.invoice.term") || "Plazo"}
							field={form}
							type="number"
							placeholder={textGet("billing.invoice.term.placeholder") || ""}
						/>
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardHeader className="flex flex-row justify-between items-center">
					<div>
						<CardTitle>
							<Text uuid="billing.invoice.items" />
						</CardTitle>
						<CardDescription>
							<Text uuid="billing.invoice.items.desc" />
						</CardDescription>
					</div>
					<div>
						<Button
							type="button"
							variant="outline"
							size="sm"
							onClick={() =>
								append({
									description: "",
									quantity: 1,
									unit_price: 0,
									discount: 0,
									tax_rate: "0",
									ice_tax: 0,
								})
							}
						>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="billing.invoice.item.add" />
						</Button>
					</div>
				</CardHeader>
				<CardContent className="space-y-4">
					{fields.map((field, index) => (
						<div
							key={field.id}
							className="grid lg:grid-cols-[1fr_3fr_2fr_2fr_1fr] gap-2 items-end border p-4 rounded-md"
						>
							<div>
								<FormInput
									type="number"
									field={form}
									name={`items.${index}.quantity` as const}
									label={textGet("billing.invoice.item.qty") || "Cant."}
								/>
							</div>
							<div>
								<FormInput
									field={form}
									name={`items.${index}.description` as const}
									label={
										textGet("billing.invoice.item.description") || "Descripción"
									}
								/>
							</div>
							<div>
								<FormInput
									type="number"
									step="0.01"
									field={form}
									name={`items.${index}.unit_price` as const}
									label={textGet("billing.invoice.item.price") || "Precio"}
								/>
							</div>
							<div>
								<FormSelect
									field={form}
									name={`items.${index}.tax_rate` as const}
									label={textGet("billing.invoice.item.tax") || "Impuesto"}
									options={[
										{ label: "0%", value: "0" },
										{ label: "12%", value: "2" },
										{ label: "15%", value: "4" },
									]}
								/>
							</div>
							<div className="flex justify-end pb-2">
								<Button
									type="button"
									variant="destructive"
									size="icon"
									onClick={() => remove(index)}
									disabled={fields.length === 1}
								>
									<Trash className="h-4 w-4" />
								</Button>
							</div>
						</div>
					))}
				</CardContent>
			</Card>

			<Card>
				<CardHeader>
					<CardTitle>
						<Text uuid="billing.invoice.summary.title" />
					</CardTitle>
					<CardDescription>
						<Text uuid="billing.invoice.summary.desc" />
					</CardDescription>
				</CardHeader>
				<CardContent>
					<div className="w-full flex justify-end">
						<div className="w-full md:w-1/3 space-y-2">
							<div className="flex justify-between text-muted-foreground">
								<span>
									<Text uuid="billing.invoice.subtotal" />
								</span>
								<span>${subtotal.toFixed(2)}</span>
							</div>
							<div className="flex justify-between text-muted-foreground">
								<span>
									<Text uuid="billing.invoice.discount" />
								</span>
								<span>-${discountTotal.toFixed(2)}</span>
							</div>
							<div className="flex justify-between text-muted-foreground">
								<span>
									<Text uuid="billing.invoice.tax" />
								</span>
								<span>${totalTax.toFixed(2)}</span>
							</div>
							<div className="flex justify-between text-lg font-bold border-t pt-2">
								<span>
									<Text uuid="billing.invoice.total" />
								</span>
								<span>${total.toFixed(2)}</span>
							</div>
						</div>
					</div>
				</CardContent>
			</Card>

			<div className="flex justify-end gap-2">
				<Button type="button" variant="outline" onClick={() => navigate(-1)}>
					<Text uuid="form.cancel" />
				</Button>
				<Button type="submit" disabled={loading}>
					{loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
					<Text uuid="billing.invoice.save" />
				</Button>
			</div>
		</div>
	);
}
