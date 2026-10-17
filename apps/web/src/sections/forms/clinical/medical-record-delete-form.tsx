import { useText } from "@pengi/shared";
import { Button, Form, FormTextArea } from "@pengi/ui";
import { Loader2, Trash2 } from "lucide-react";
import { z } from "zod";

const deleteSchema = z.object({
	reason: z
		.string()
		.trim()
		.min(1, "clinical.medical_record.delete.error.reason_required")
		.max(500, "clinical.medical_record.delete.error.reason_long"),
});

export type MedicalRecordDeleteValues = z.infer<typeof deleteSchema>;

interface MedicalRecordDeleteFormProps {
	loading?: boolean;
	onSubmit: (values: MedicalRecordDeleteValues) => void;
}

/** Asks for the reason (required) before a medical record is deleted. */
export default function MedicalRecordDeleteForm({
	loading,
	onSubmit,
}: MedicalRecordDeleteFormProps) {
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
						label={textGet("clinical.medical_record.delete.reason")}
						placeholder={textGet(
							"clinical.medical_record.delete.reason.placeholder",
						)}
					/>
					<div className="flex justify-end">
						<Button type="submit" variant="destructive" disabled={loading}>
							{loading ? (
								<Loader2 className="mr-2 h-4 w-4 animate-spin" />
							) : (
								<Trash2 className="mr-2 h-4 w-4" />
							)}
							{textGet("clinical.medical_record.delete.confirm")}
						</Button>
					</div>
				</div>
			)}
		</Form>
	);
}
