import { toDateOnlyString, useText } from "@pengi/shared";
import {
	Button,
	Checkbox,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
	Field,
	FieldDescription,
	FieldError,
	FieldLabel,
	Form,
	FormTextArea,
} from "@pengi/ui";
import { Loader2, UploadCloud } from "lucide-react";
import React from "react";
import { Controller } from "react-hook-form";
import { z } from "zod";
import type { ExamOrder, ExamResultUpload } from "@/api/exam-order-service";
import { toastBatchResult } from "@/components/features/patient-attachments/attachment-actions";
import { AttachmentFilesPicker } from "@/components/features/patient-attachments/attachment-files-picker";
import type { UploadItem } from "@/components/features/patient-attachments/upload-batch";
import { UploadProgressList } from "@/components/features/patient-attachments/upload-progress-list";
import { FormCalendar } from "@/components/forms/form-calendar";
import { itemHasResult } from "@/lib/exam-orders";
import { uploadExamResults } from "./upload-exam-results";

const uploadSchema = z.object({
	item_ids: z
		.array(z.number())
		.min(1, "clinical.exam_orders.results.error.items_required"),
	files: z
		.array(z.instanceof(File))
		.min(1, "clinical.attachment.form.error.file_required"),
	taken_at: z.date().optional(),
	description: z
		.string()
		.max(255, "clinical.attachment.form.error.description_long")
		.optional(),
});

type UploadValues = z.infer<typeof uploadSchema>;

interface ExamResultsUploadDialogProps {
	order: ExamOrder;
	open: boolean;
	/** Exams to preselect; defaults to those without a result. */
	initialItemIds?: number[];
	onOpenChange: (open: boolean) => void;
	/** Called with the last upload's order (status recomputed). */
	onUploaded: (order: ExamOrder) => void;
}

/**
 * Uploads result files while marking which exams they cover. Each file is
 * one request carrying the selected exams; progress, HEIC conversion and the
 * duplicate warning come from the attachments batch.
 */
export function ExamResultsUploadDialog({
	order,
	open,
	initialItemIds,
	onOpenChange,
	onUploaded,
}: ExamResultsUploadDialogProps) {
	const { textGet } = useText();
	const [uploading, setUploading] = React.useState(false);
	const [batch, setBatch] = React.useState<UploadItem[] | null>(null);

	const defaultItemIds =
		initialItemIds ??
		order.items.filter((item) => !itemHasResult(item)).map((item) => item.ID);

	async function handleSubmit(values: UploadValues) {
		setUploading(true);
		setBatch(
			values.files.map((file, id) => ({
				id,
				name: file.name,
				status: "pending",
				progress: 0,
			})),
		);
		try {
			const result = await uploadExamResults(
				order.ID,
				values.files,
				{
					item_ids: values.item_ids,
					taken_at: values.taken_at
						? toDateOnlyString(values.taken_at)
						: undefined,
					description: values.description?.trim() || undefined,
				},
				(id, patch) =>
					setBatch(
						(items) =>
							items?.map((item) =>
								item.id === id ? { ...item, ...patch } : item,
							) ?? null,
					),
			);
			toastBatchResult(result, textGet);
			const last: ExamResultUpload | undefined = result.results.at(-1);
			if (last?.order) onUploaded(last.order);
		} finally {
			setUploading(false);
		}
	}

	function close(next: boolean) {
		if (next || uploading) return;
		onOpenChange(false);
		setBatch(null);
	}

	return (
		<Dialog open={open} onOpenChange={close}>
			<DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
				<DialogHeader>
					<DialogTitle>
						{textGet("clinical.exam_orders.results.upload.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet("clinical.exam_orders.results.upload.description")}
					</DialogDescription>
				</DialogHeader>
				{batch ? (
					<div className="space-y-4">
						<UploadProgressList items={batch} />
						<div className="flex justify-end">
							<Button
								type="button"
								disabled={uploading}
								onClick={() => close(false)}
							>
								{textGet("clinical.exam_orders.results.upload.done")}
							</Button>
						</div>
					</div>
				) : (
					<Form
						schema={uploadSchema}
						onSubmit={handleSubmit}
						defaultValues={{
							item_ids: defaultItemIds,
							files: [],
							description: "",
						}}
					>
						{(field) => (
							<div className="space-y-4">
								<Controller
									control={field.control}
									name="item_ids"
									render={({ field: idsField, fieldState }) => {
										const ids: number[] = idsField.value ?? [];
										const idSet = new Set(ids);
										return (
											<Field data-invalid={fieldState.invalid}>
												<FieldLabel>
													{textGet("clinical.exam_orders.results.covers")}
												</FieldLabel>
												<ul className="max-h-48 space-y-1.5 overflow-y-auto rounded-md border p-2">
													{order.items.map((item) => {
														const id = `result-item-${item.ID}`;
														return (
															<li
																key={item.ID}
																className="flex items-center gap-2"
															>
																<Checkbox
																	id={id}
																	checked={idSet.has(item.ID)}
																	onCheckedChange={(checked) =>
																		idsField.onChange(
																			checked === true
																				? [...ids, item.ID]
																				: ids.filter((v) => v !== item.ID),
																		)
																	}
																/>
																<label
																	htmlFor={id}
																	className="cursor-pointer text-sm"
																>
																	{item.name}
																</label>
															</li>
														);
													})}
												</ul>
												{fieldState.invalid && (
													<FieldError errors={[fieldState.error]} />
												)}
											</Field>
										);
									}}
								/>
								<Controller
									control={field.control}
									name="files"
									render={({ field: filesField, fieldState }) => (
										<Field data-invalid={fieldState.invalid}>
											<FieldLabel htmlFor="exam-result-file">
												{textGet("clinical.attachment.form.file")}
											</FieldLabel>
											<AttachmentFilesPicker
												id="exam-result-file"
												files={filesField.value ?? []}
												onChange={filesField.onChange}
											/>
											<FieldDescription>
												{textGet("clinical.attachment.form.subtitle")}
											</FieldDescription>
											{fieldState.invalid && (
												<FieldError errors={[fieldState.error]} />
											)}
										</Field>
									)}
								/>
								<FormCalendar
									field={field}
									name="taken_at"
									label={textGet("clinical.attachment.form.taken_at")}
									description={textGet(
										"clinical.attachment.form.taken_at.description",
									)}
									isOptional
								/>
								<FormTextArea
									field={field}
									name="description"
									label={textGet("clinical.attachment.form.description")}
									placeholder={textGet(
										"clinical.attachment.form.description.placeholder",
									)}
									isOptional
								/>
								<div className="flex justify-end">
									<Button type="submit" disabled={uploading}>
										{uploading ? (
											<Loader2 className="mr-2 h-4 w-4 animate-spin" />
										) : (
											<UploadCloud className="mr-2 h-4 w-4" />
										)}
										{textGet("clinical.attachment.form.submit")}
									</Button>
								</div>
							</div>
						)}
					</Form>
				)}
			</DialogContent>
		</Dialog>
	);
}
