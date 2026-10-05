import { parseDateOnly, useText } from "@pengi/shared";
import {
	Button,
	Field,
	FieldDescription,
	FieldError,
	FieldLabel,
	Form,
	FormSelect,
	FormTextArea,
} from "@pengi/ui";
import { Loader2, Save, Trash2, UploadCloud } from "lucide-react";
import { Controller, type UseFormReturn } from "react-hook-form";
import { z } from "zod";
import {
	ATTACHMENT_CATEGORIES,
	type PatientAttachment,
} from "@/api/patient-attachment-service";
import { AttachmentFilesPicker } from "@/components/features/patient-attachments/attachment-files-picker";
import { FormCalendar } from "@/components/forms/form-calendar";

const attachmentSchema = z.object({
	// Size and type are checked per file (attachmentFileError), so one bad file
	// doesn't block the rest.
	files: z
		.array(z.instanceof(File))
		.min(1, "clinical.attachment.form.error.file_required"),
	category: z.enum(ATTACHMENT_CATEGORIES, {
		error: "clinical.attachment.form.error.category",
	}),
	taken_at: z.date().optional(),
	description: z
		.string()
		.max(255, "clinical.attachment.form.error.description_long")
		.optional(),
});

const editSchema = attachmentSchema.omit({ files: true });

export type PatientAttachmentFormValues = z.infer<typeof attachmentSchema>;
export type PatientAttachmentEditValues = z.infer<typeof editSchema>;

// Upload and edit forms have different value shapes; the shared fields only
// touch category, taken_at and description.
// biome-ignore lint/suspicious/noExplicitAny: shared by two form value shapes
type MetadataFormMethods = UseFormReturn<any, unknown, any>;

/** Category, exam date and description: what upload and edit have in common. */
function AttachmentMetadataFields({ field }: { field: MetadataFormMethods }) {
	const { textGet } = useText();
	return (
		<>
			<FormSelect
				field={field}
				name="category"
				label={textGet("clinical.attachment.form.category")}
				placeholder={textGet("clinical.attachment.form.category.placeholder")}
				options={ATTACHMENT_CATEGORIES.map((category) => ({
					value: category,
					label: textGet(`clinical.attachment.category.${category}`),
				}))}
			/>
			<FormCalendar
				field={field}
				name="taken_at"
				label={textGet("clinical.attachment.form.taken_at")}
				description={textGet("clinical.attachment.form.taken_at.description")}
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
		</>
	);
}

interface PatientAttachmentFormProps {
	loading?: boolean;
	onSubmit: (values: PatientAttachmentFormValues) => void;
}

/** One or more files sharing a category, optional exam date and optional description. */
export default function PatientAttachmentForm({
	loading,
	onSubmit,
}: PatientAttachmentFormProps) {
	const { textGet } = useText();

	return (
		<Form
			schema={attachmentSchema}
			onSubmit={onSubmit}
			defaultValues={{ description: "" }}
		>
			{(field) => (
				<div className="space-y-4">
					<Controller
						control={field.control}
						name="files"
						render={({ field: filesField, fieldState }) => (
							<Field data-invalid={fieldState.invalid}>
								<FieldLabel htmlFor="attachment-file">
									{textGet("clinical.attachment.form.file")}
								</FieldLabel>
								<AttachmentFilesPicker
									id="attachment-file"
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
					<AttachmentMetadataFields
						field={field as unknown as MetadataFormMethods}
					/>
					<div className="flex justify-end">
						<Button type="submit" disabled={loading}>
							{loading ? (
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
	);
}

interface PatientAttachmentEditFormProps {
	attachment: PatientAttachment;
	loading?: boolean;
	onSubmit: (values: PatientAttachmentEditValues) => void;
}

/** Edits an attachment's metadata; the file itself can't be replaced. */
export function PatientAttachmentEditForm({
	attachment,
	loading,
	onSubmit,
}: PatientAttachmentEditFormProps) {
	const { textGet } = useText();

	return (
		<Form
			schema={editSchema}
			onSubmit={onSubmit}
			defaultValues={{
				category: attachment.category,
				taken_at: parseDateOnly(attachment.taken_at),
				description: attachment.description ?? "",
			}}
		>
			{(field) => (
				<div className="space-y-4">
					<AttachmentMetadataFields
						field={field as unknown as MetadataFormMethods}
					/>
					<div className="flex justify-end">
						<Button type="submit" disabled={loading}>
							{loading ? (
								<Loader2 className="mr-2 h-4 w-4 animate-spin" />
							) : (
								<Save className="mr-2 h-4 w-4" />
							)}
							{textGet("clinical.attachment.form.save")}
						</Button>
					</div>
				</div>
			)}
		</Form>
	);
}

const deleteSchema = z.object({
	reason: z
		.string()
		.trim()
		.min(1, "clinical.attachment.delete.error.reason_required")
		.max(500, "clinical.attachment.delete.error.reason_long"),
});

export type PatientAttachmentDeleteValues = z.infer<typeof deleteSchema>;

interface PatientAttachmentDeleteFormProps {
	loading?: boolean;
	onSubmit: (values: PatientAttachmentDeleteValues) => void;
}

/** Asks for the reason (required) before an attachment is deleted. */
export function PatientAttachmentDeleteForm({
	loading,
	onSubmit,
}: PatientAttachmentDeleteFormProps) {
	const { textGet } = useText();

	return (
		<Form
			schema={deleteSchema}
			onSubmit={onSubmit}
			defaultValues={{ reason: "" }}
		>
			{(field) => (
				<div className="space-y-4">
					<FormTextArea
						field={field}
						name="reason"
						label={textGet("clinical.attachment.delete.reason")}
						placeholder={textGet(
							"clinical.attachment.delete.reason.placeholder",
						)}
					/>
					<div className="flex justify-end">
						<Button type="submit" variant="destructive" disabled={loading}>
							{loading ? (
								<Loader2 className="mr-2 h-4 w-4 animate-spin" />
							) : (
								<Trash2 className="mr-2 h-4 w-4" />
							)}
							{textGet("clinical.attachment.delete.confirm")}
						</Button>
					</div>
				</div>
			)}
		</Form>
	);
}
