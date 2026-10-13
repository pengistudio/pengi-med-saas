import { useText } from "@pengi/shared";
import {
	Badge,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Field,
	FieldError,
	Form,
	FormInput,
	FormSelect,
	FormTextArea,
} from "@pengi/ui";
import { Loader2, Save } from "lucide-react";
import { Controller } from "react-hook-form";
import { z } from "zod";
import {
	EXAM_PRIORITIES,
	type ExamCatalogItem,
	type ExamOrder,
	type ExamProfile,
} from "@/api/exam-order-service";
import { FormDoctorSelect } from "@/components/features/doctors/doctor-select";
import { ExamPicker } from "@/components/features/exam-orders/exam-picker";
import { SelectedExamsList } from "@/components/features/exam-orders/selected-exams-list";
import { FormIcd11Select } from "@/components/forms/form-icd11-select";
import {
	type DraftExamItem,
	draftsFromOrder,
	EXAM_PRIORITY_KEYS,
	hasAnyResult,
} from "@/lib/exam-orders";

const examOrderSchema = z.object({
	items: z
		.array(z.custom<DraftExamItem>())
		.min(1, "clinical.exam_orders.form.error.items_required"),
	diagnoses: z.array(z.object({ code: z.string(), title: z.string() })),
	priority: z.enum(EXAM_PRIORITIES, {
		error: "clinical.exam_orders.form.error.priority",
	}),
	destination_lab: z
		.string()
		.max(255, "clinical.exam_orders.form.error.too_long")
		.optional(),
	notes: z
		.string()
		.max(2000, "clinical.exam_orders.form.error.too_long")
		.optional(),
	doctor_id: z.number().nullable().optional(),
});

export type ExamOrderFormValues = z.infer<typeof examOrderSchema>;

interface ExamOrderFormProps {
	catalog: ExamCatalogItem[];
	profiles: ExamProfile[];
	/** The order being edited; absent on create. */
	order?: ExamOrder;
	loading?: boolean;
	/** The patient's médico de cabecera, a default for the doctor on create. */
	patientDoctorId?: number | null;
	onSubmit: (values: ExamOrderFormValues) => void;
	onCancel: () => void;
}

/**
 * Creates or edits an exam order. Once the order has results, its header and
 * existing exams are read-only: only new exams can be added (the server
 * rejects any other change).
 */
export default function ExamOrderForm({
	catalog,
	profiles,
	order,
	loading,
	patientDoctorId,
	onSubmit,
	onCancel,
}: ExamOrderFormProps) {
	const { textGet } = useText();
	const locked = order ? hasAnyResult(order) : false;

	return (
		<Form
			schema={examOrderSchema}
			onSubmit={onSubmit}
			defaultValues={{
				items: order ? draftsFromOrder(order) : [],
				diagnoses: order?.diagnoses ?? [],
				priority: order?.priority ?? "routine",
				destination_lab: order?.destination_lab ?? "",
				notes: order?.notes ?? "",
				doctor_id: order?.doctor_id ?? null,
			}}
		>
			{(field) => (
				<div className="grid gap-4 lg:grid-cols-[1fr_22rem]">
					<Card>
						<CardHeader>
							<CardTitle>
								{textGet("clinical.exam_orders.form.exams")}
							</CardTitle>
							<CardDescription>
								{locked
									? textGet("clinical.exam_orders.form.locked")
									: textGet("clinical.exam_orders.form.exams.description")}
							</CardDescription>
						</CardHeader>
						<CardContent>
							<Controller
								control={field.control}
								name="items"
								render={({ field: itemsField, fieldState }) => (
									<Field data-invalid={fieldState.invalid} className="gap-4">
										<ExamPicker
											catalog={catalog}
											profiles={profiles}
											value={itemsField.value ?? []}
											onChange={itemsField.onChange}
										/>
										<div className="space-y-2">
											<p className="flex items-center gap-2 text-sm font-medium">
												{textGet("clinical.exam_orders.form.selected")}
												<Badge variant="secondary">
													{(itemsField.value ?? []).length}
												</Badge>
											</p>
											<SelectedExamsList
												value={itemsField.value ?? []}
												onChange={itemsField.onChange}
											/>
										</div>
										{fieldState.invalid && (
											<FieldError errors={[fieldState.error]} />
										)}
									</Field>
								)}
							/>
						</CardContent>
					</Card>

					<div className="space-y-4">
						<Card>
							<CardHeader>
								<CardTitle>
									{textGet("clinical.exam_orders.form.details")}
								</CardTitle>
							</CardHeader>
							<CardContent className="space-y-4">
								{locked && order ? (
									<LockedHeader order={order} />
								) : (
									<>
										<FormDoctorSelect
											field={field}
											name="doctor_id"
											autoDefault={!order}
											fallbacks={[patientDoctorId]}
										/>
										<FormIcd11Select
											field={field}
											name="diagnoses"
											label={textGet("clinical.exam_orders.field.diagnoses")}
											isOptional
											system="cie10"
										/>
										<FormSelect
											field={field}
											name="priority"
											label={textGet("clinical.exam_orders.field.priority")}
											options={EXAM_PRIORITIES.map((priority) => ({
												value: priority,
												label: textGet(EXAM_PRIORITY_KEYS[priority]),
											}))}
										/>
										<FormInput
											field={field}
											name="destination_lab"
											label={textGet(
												"clinical.exam_orders.field.destination_lab",
											)}
											placeholder={textGet(
												"clinical.exam_orders.field.destination_lab.placeholder",
											)}
											isOptional
										/>
										<FormTextArea
											field={field}
											name="notes"
											label={textGet("clinical.exam_orders.field.notes")}
											placeholder={textGet(
												"clinical.exam_orders.field.notes.placeholder",
											)}
											isOptional
										/>
									</>
								)}
							</CardContent>
						</Card>
						<div className="flex justify-end gap-2">
							<Button type="button" variant="outline" onClick={onCancel}>
								{textGet("form.cancel")}
							</Button>
							<Button type="submit" disabled={loading}>
								{loading ? (
									<Loader2 className="mr-2 h-4 w-4 animate-spin" />
								) : (
									<Save className="mr-2 h-4 w-4" />
								)}
								{textGet("clinical.exam_orders.form.submit")}
							</Button>
						</div>
					</div>
				</div>
			)}
		</Form>
	);
}

/** The header of an order with results, which can no longer change. */
function LockedHeader({ order }: { order: ExamOrder }) {
	const { textGet } = useText();
	const diagnoses = order.diagnoses ?? [];
	return (
		<dl className="space-y-3 text-sm">
			<div>
				<dt className="text-muted-foreground">
					{textGet("clinical.exam_orders.field.diagnoses")}
				</dt>
				<dd>
					{diagnoses.length > 0
						? diagnoses.map((d) => `${d.code} ${d.title}`).join("; ")
						: "—"}
				</dd>
			</div>
			<div>
				<dt className="text-muted-foreground">
					{textGet("clinical.exam_orders.field.priority")}
				</dt>
				<dd>{textGet(EXAM_PRIORITY_KEYS[order.priority] ?? order.priority)}</dd>
			</div>
			<div>
				<dt className="text-muted-foreground">
					{textGet("clinical.exam_orders.field.destination_lab")}
				</dt>
				<dd>{order.destination_lab || "—"}</dd>
			</div>
			<div>
				<dt className="text-muted-foreground">
					{textGet("clinical.exam_orders.field.notes")}
				</dt>
				<dd className="whitespace-pre-wrap">{order.notes || "—"}</dd>
			</div>
		</dl>
	);
}

const voidSchema = z.object({
	reason: z
		.string()
		.trim()
		.min(1, "clinical.exam_orders.void.error.reason_required")
		.max(500, "clinical.exam_orders.void.error.reason_long"),
});

export type ExamOrderVoidValues = z.infer<typeof voidSchema>;

/** Asks for the reason (required) before an order is voided. */
export function ExamOrderVoidForm({
	loading,
	onSubmit,
}: {
	loading?: boolean;
	onSubmit: (values: ExamOrderVoidValues) => void;
}) {
	const { textGet } = useText();
	return (
		<Form
			schema={voidSchema}
			onSubmit={onSubmit}
			defaultValues={{ reason: "" }}
		>
			{(field) => (
				<div className="space-y-4">
					<FormTextArea
						field={field}
						name="reason"
						label={textGet("clinical.exam_orders.void.reason")}
						placeholder={textGet(
							"clinical.exam_orders.void.reason.placeholder",
						)}
					/>
					<div className="flex justify-end">
						<Button type="submit" variant="destructive" disabled={loading}>
							{loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
							{textGet("clinical.exam_orders.void.confirm")}
						</Button>
					</div>
				</div>
			)}
		</Form>
	);
}
