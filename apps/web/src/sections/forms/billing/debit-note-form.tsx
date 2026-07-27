import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Field,
	FieldLabel,
	FormInput,
	FormSelect,
	Text,
} from "@pengi/ui";
import { Loader2, Plus, Trash } from "lucide-react";
import { useEffect, useState } from "react";
import { type UseFormReturn, useFieldArray } from "react-hook-form";
import { useNavigate } from "react-router";
import * as z from "zod";
import {
	createDebitNote,
	getAllInvoices,
	type Invoice,
} from "@/api/billing-service";
import { Form } from "@/components/forms/form";
import { useText } from "@/hooks/use-text";
import {
	IVA_PERCENTAGE_CODES,
	IVA_PERCENTAGE_CODES_AS_NUMBER,
	type TaxPercentageCode,
} from "@/lib/constants";

const debitNoteMotiveSchema = z.object({
	reason: z.string().min(1, "validation.required"),
	value: z.number().min(0.01, "validation.min_0"),
	tax_percentage_code: z.string().min(1, "validation.required"),
});

const formSchema = z.object({
	invoice_id: z.string().min(1, "validation.required"),
	motives: z.array(debitNoteMotiveSchema).min(1, "validation.min_1_item"),
});

export function DebitNoteForm() {
	const [invoices, setInvoices] = useState<Invoice[]>([]);
	const [loading, setLoading] = useState(false);
	const navigate = useNavigate();

	useEffect(() => {
		getAllInvoices({ limit: 100 }).then((res) => {
			if (res.success && res.data) {
				setInvoices(
					res.data.items.filter((inv) => inv.status === "authorized"),
				);
			}
		});
	}, []);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoading(true);

		const payload = {
			invoice_id: parseInt(values.invoice_id, 10),
			motives: values.motives.map((motive) => ({
				reason: motive.reason,
				value: motive.value,
				tax_code: "2",
				tax_percentage_code: motive.tax_percentage_code,
				tax_rate:
					IVA_PERCENTAGE_CODES_AS_NUMBER[
						motive.tax_percentage_code as TaxPercentageCode
					] || 0,
			})),
		};

		try {
			const res = await createDebitNote(payload);
			if (res.success) {
				navigate("/billing/debit-notes");
			}
		} finally {
			setLoading(false);
		}
	}

	return (
		<Form
			schema={formSchema}
			onSubmit={onSubmit}
			defaultValues={{
				invoice_id: "",
				motives: [{ reason: "", value: 0, tax_percentage_code: "0" }],
			}}
		>
			{(form) => (
				<DebitNoteFormInner form={form} invoices={invoices} loading={loading} />
			)}
		</Form>
	);
}

function DebitNoteFormInner({
	form,
	invoices,
	loading,
}: {
	// biome-ignore lint/suspicious/noExplicitAny: UseFormReturn<any> is acceptable here
	form: UseFormReturn<any>;
	invoices: Invoice[];
	loading: boolean;
}) {
	const { fields, append, remove } = useFieldArray({
		name: "motives",
		control: form.control,
	});
	const { textGet } = useText();
	const navigate = useNavigate();

	const motivesWatch: z.infer<typeof debitNoteMotiveSchema>[] =
		form.watch("motives");

	const subtotal = (motivesWatch || []).reduce(
		(sum, motive) => sum + (motive.value || 0),
		0,
	);
	const totalTax = (motivesWatch || []).reduce((sum, motive) => {
		const rate =
			IVA_PERCENTAGE_CODES_AS_NUMBER[
				motive.tax_percentage_code as TaxPercentageCode
			] || 0;
		return sum + motive.value * rate;
	}, 0);
	const total = subtotal + totalTax;

	const taxOptions = Object.entries(IVA_PERCENTAGE_CODES).map(
		([value, label]) => ({ label, value }),
	);

	return (
		<div className="space-y-6">
			<Card>
				<CardHeader>
					<CardTitle>
						<Text uuid="billing.debit_note.create.title" />
					</CardTitle>
					<CardDescription>
						<Text uuid="billing.debit_note.create.description" />
					</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					<FormSelect
						name="invoice_id"
						label={textGet("billing.debit_note.invoice")}
						placeholder={textGet("billing.debit_note.invoice.placeholder")}
						emptyMessage={textGet("billing.debit_note.invoice.empty")}
						field={form}
						options={invoices.map((inv) => ({
							label: `${inv.establishment_code}-${inv.emission_point_code}-${inv.sequential}${
								inv.patient
									? ` — ${inv.patient.first_name} ${inv.patient.last_name}`
									: ""
							}`,
							value: inv.ID.toString(),
						}))}
					/>
				</CardContent>
			</Card>

			<Card>
				<CardHeader className="flex flex-row justify-between items-center">
					<div>
						<CardTitle>
							<Text uuid="billing.debit_note.motives" />
						</CardTitle>
					</div>
					<div className="flex items-center gap-4">
						<span className="text-sm font-medium whitespace-nowrap">
							<Text uuid="billing.invoice.total" />: ${total.toFixed(2)}
						</span>
						<Button
							type="button"
							variant="outline"
							size="sm"
							onClick={() =>
								append({ reason: "", value: 0, tax_percentage_code: "0" })
							}
						>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="billing.debit_note.motive.add" />
						</Button>
					</div>
				</CardHeader>
				<CardContent className="space-y-4">
					{fields.map((field, index) => {
						const motive = motivesWatch?.[index];
						const rate = motive
							? IVA_PERCENTAGE_CODES_AS_NUMBER[
									motive.tax_percentage_code as TaxPercentageCode
								] || 0
							: 0;
						const motiveValue = motive?.value || 0;
						const lineTotal = motiveValue + motiveValue * rate;

						return (
							<div
								key={field.id}
								className="grid lg:grid-cols-[3fr_2fr_2fr_2fr_1fr] gap-2 items-end border p-4 rounded-md"
							>
								<div>
									<FormInput
										field={form}
										name={`motives.${index}.reason` as const}
										label={textGet("billing.debit_note.motive.reason")}
										placeholder={textGet(
											"billing.debit_note.motive.reason.placeholder",
										)}
									/>
								</div>
								<div>
									<FormInput
										type="number"
										step="0.01"
										field={form}
										name={`motives.${index}.value` as const}
										label={textGet("billing.debit_note.motive.value")}
									/>
								</div>
								<div>
									<FormSelect
										field={form}
										name={`motives.${index}.tax_percentage_code` as const}
										label={textGet("billing.debit_note.motive.tax")}
										options={taxOptions}
									/>
								</div>
								<div>
									<Field className="flex flex-col h-full justify-end">
										<FieldLabel className="mb-2">
											{textGet("billing.invoice.item.total")}
										</FieldLabel>
										<div className="h-9 flex items-center px-3 text-sm font-medium border rounded-md bg-muted/30">
											${lineTotal.toFixed(2)}
										</div>
									</Field>
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
						);
					})}
				</CardContent>
			</Card>

			<Card>
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
					<Text uuid="billing.debit_note.save" />
				</Button>
			</div>
		</div>
	);
}
