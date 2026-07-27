import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Combobox,
	ComboboxContent,
	ComboboxEmpty,
	ComboboxInput,
	ComboboxItem,
	ComboboxList,
	Field,
	FieldLabel,
	FormInput,
	FormSelect,
	FormTextArea,
	Text,
} from "@pengi/ui";
import { Loader2, Plus, Trash } from "lucide-react";
import { useEffect, useState } from "react";
import { type UseFormReturn, useFieldArray } from "react-hook-form";
import { useNavigate } from "react-router";
import * as z from "zod";
import {
	type CatalogItem,
	createCreditNote,
	getAllCatalogItems,
	getAllInvoices,
	type Invoice,
} from "@/api/billing-service";
import { Form } from "@/components/forms/form";
import { useText } from "@/hooks/use-text";
import {
	IVA_PERCENTAGE_CODES_AS_NUMBER,
	type TaxPercentageCode,
} from "@/lib/constants";

const creditNoteItemSchema = z.object({
	product_id: z.string().min(1, "validation.required"),
	description: z.string().min(1, "validation.required"),
	quantity: z.number().min(1, "validation.min_1"),
	unit_price: z.number().min(0, "validation.min_0"),
	tax_rate: z.string().min(1, "validation.required"),
});

const formSchema = z.object({
	invoice_id: z.string().min(1, "validation.required"),
	reason: z.string().min(1, "validation.required"),
	items: z.array(creditNoteItemSchema).min(1, "validation.min_1_item"),
});

export function CreditNoteForm() {
	const [invoices, setInvoices] = useState<Invoice[]>([]);
	const [catalogItems, setCatalogItems] = useState<CatalogItem[]>([]);
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
		getAllCatalogItems({ limit: 100 }).then((res) => {
			if (res.success && res.data) {
				setCatalogItems(res.data.items);
			}
		});
	}, []);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoading(true);

		const payload = {
			invoice_id: parseInt(values.invoice_id, 10),
			reason: values.reason,
			items: values.items.map((item) => ({
				product_id: parseInt(item.product_id, 10),
				quantity: item.quantity,
			})),
		};

		try {
			const res = await createCreditNote(payload);
			if (res.success) {
				navigate("/billing/credit-notes");
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
				reason: "",
				items: [
					{
						product_id: undefined,
						description: undefined,
						quantity: 1,
						unit_price: 0,
						tax_rate: "0",
					},
				],
			}}
		>
			{(form) => (
				<CreditNoteFormInner
					form={form}
					invoices={invoices}
					catalogItems={catalogItems}
					loading={loading}
				/>
			)}
		</Form>
	);
}

function CreditNoteFormInner({
	form,
	invoices,
	catalogItems,
	loading,
}: {
	// biome-ignore lint/suspicious/noExplicitAny: UseFormReturn<any> is acceptable here
	form: UseFormReturn<any>;
	invoices: Invoice[];
	catalogItems: CatalogItem[];
	loading: boolean;
}) {
	const { fields, append, remove } = useFieldArray({
		name: "items",
		control: form.control,
	});
	const { textGet } = useText();
	const navigate = useNavigate();
	const [itemInputValues, setItemInputValues] = useState<string[]>([]);

	const itemsWatch: z.infer<typeof creditNoteItemSchema>[] =
		form.watch("items");

	const subtotal = (itemsWatch || []).reduce(
		(sum, item) => sum + item.quantity * item.unit_price,
		0,
	);
	const totalTax = (itemsWatch || []).reduce((sum, item) => {
		const rate =
			IVA_PERCENTAGE_CODES_AS_NUMBER[item.tax_rate as TaxPercentageCode] || 0;
		return sum + item.quantity * item.unit_price * rate;
	}, 0);
	const total = subtotal + totalTax;

	return (
		<div className="space-y-6">
			<Card>
				<CardHeader>
					<CardTitle>
						<Text uuid="billing.credit_note.create.title" />
					</CardTitle>
					<CardDescription>
						<Text uuid="billing.credit_note.create.description" />
					</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					<FormSelect
						name="invoice_id"
						label={textGet("billing.credit_note.invoice")}
						placeholder={textGet("billing.credit_note.invoice.placeholder")}
						emptyMessage={textGet("billing.credit_note.invoice.empty")}
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
					<FormTextArea
						field={form}
						name="reason"
						label={textGet("billing.credit_note.reason")}
						placeholder={textGet("billing.credit_note.reason.placeholder")}
					/>
				</CardContent>
			</Card>

			<Card>
				<CardHeader className="flex flex-row justify-between items-center">
					<div>
						<CardTitle>
							<Text uuid="billing.invoice.items" />
						</CardTitle>
					</div>
					<div>
						<Button
							type="button"
							variant="outline"
							size="sm"
							onClick={() =>
								append({
									product_id: undefined,
									description: "",
									quantity: 1,
									unit_price: 0,
									tax_rate: "0",
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
									label={textGet("billing.invoice.item.qty")}
								/>
							</div>
							<div>
								<Field className="flex flex-col h-full justify-end">
									<FieldLabel className="mb-2">
										{textGet("billing.invoice.item.description")}
									</FieldLabel>
									<Combobox
										value={form.watch(`items.${index}.description` as const)}
										onValueChange={(val) => {
											if (!val) return;
											form.setValue(
												`items.${index}.description` as const,
												val as string,
											);
											setItemInputValues((prev) => {
												const next = [...prev];
												next[index] = val as string;
												return next;
											});

											const selectedItem = catalogItems.find(
												(item) => item.name === val,
											);
											if (selectedItem) {
												form.setValue(
													`items.${index}.product_id` as const,
													selectedItem.ID.toString(),
												);
												form.setValue(
													`items.${index}.unit_price` as const,
													selectedItem.unit_price,
												);
												form.setValue(
													`items.${index}.tax_rate` as const,
													selectedItem.tax_percentage_code,
												);
											}
										}}
										onInputValueChange={(val) => {
											form.setValue(`items.${index}.description` as const, val);
											setItemInputValues((prev) => {
												const next = [...prev];
												next[index] = val;
												return next;
											});
										}}
										filteredItems={catalogItems
											.filter((item) => {
												const q = (itemInputValues[index] ?? "").toLowerCase();
												return !q || item.name.toLowerCase().includes(q);
											})
											.map((item) => item.name)}
									>
										<ComboboxInput
											placeholder={textGet(
												"billing.invoice.item.combobox.placeholder",
											)}
										/>
										<ComboboxContent>
											<ComboboxEmpty>
												{textGet("common.no_results")}
											</ComboboxEmpty>
											<ComboboxList>
												{catalogItems
													.filter((item) => {
														const q = (
															itemInputValues[index] ?? ""
														).toLowerCase();
														return !q || item.name.toLowerCase().includes(q);
													})
													.map((item) => (
														<ComboboxItem
															key={item.ID.toString()}
															value={item.name}
														>
															{item.name}
														</ComboboxItem>
													))}
											</ComboboxList>
										</ComboboxContent>
									</Combobox>
								</Field>
							</div>
							<div>
								<FormInput
									type="number"
									step="0.01"
									field={form}
									name={`items.${index}.unit_price` as const}
									label={textGet("billing.invoice.item.price")}
									disabled
								/>
							</div>
							<div>
								<FormSelect
									field={form}
									name={`items.${index}.tax_rate` as const}
									label={textGet("billing.invoice.item.tax")}
									disabled
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
					<Text uuid="billing.credit_note.save" />
				</Button>
			</div>
		</div>
	);
}
