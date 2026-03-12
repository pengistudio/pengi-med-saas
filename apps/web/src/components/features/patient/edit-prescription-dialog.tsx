import { Save } from "lucide-react";
import React from "react";
import { z } from "zod";
import { updatePrescription } from "@/api/clinical-service";
import { Form } from "@/components/forms/form";
import { FormTextArea } from "@/components/forms/form-textarea";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";
const prescriptionSchema = z.object({
	content: z.string({ error: "Campo requerido" }).min(1, "Campo requerido"),
	indications: z.string({ error: "Campo requerido" }).min(1, "Campo requerido"),
});

interface EditPrescriptionDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	medicalRecordId: number;
	prescription?: {
		content: string;
		indications: string;
	} | null;
	onSuccess?: () => void;
}

export default function EditPrescriptionDialog({
	open,
	onOpenChange,
	medicalRecordId,
	prescription,
	onSuccess,
}: EditPrescriptionDialogProps) {
	const [loading, setLoading] = React.useState(false);
	const { textGet } = useText();

	async function onSubmit(values: z.infer<typeof prescriptionSchema>) {
		setLoading(true);
		const res = await updatePrescription(medicalRecordId, values);
		if (res.success) {
			onOpenChange(false);
			onSuccess?.();
		}
		setLoading(false);
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[525px]">
				<DialogHeader>
					<DialogTitle>
						<Text uuid="dialog.edit_prescription.title" />
					</DialogTitle>
					<DialogDescription>
						<Text uuid="dialog.edit_prescription.description" />
					</DialogDescription>
				</DialogHeader>
				<Form
					schema={prescriptionSchema}
					defaultValues={{
						content: prescription?.content || "",
						indications: prescription?.indications || "",
					}}
					onSubmit={onSubmit}
				>
					{(field) => (
						<div className="space-y-4">
							<FormTextArea
								field={field}
								name="content"
								label={<Text uuid="dialog.edit_prescription.content" />}
								placeholder={textGet(
									"dialog.edit_prescription.content.placeholder",
								)}
							/>
							<FormTextArea
								field={field}
								name="indications"
								label={<Text uuid="dialog.edit_prescription.indications" />}
								placeholder={textGet(
									"dialog.edit_prescription.indications.placeholder",
								)}
							/>
							<DialogFooter>
								<Button type="submit" disabled={loading}>
									<Save className="mr-2 h-4 w-4" />
									<Text uuid="dialog.edit_prescription.submit" />
								</Button>
							</DialogFooter>
						</div>
					)}
				</Form>
			</DialogContent>
		</Dialog>
	);
}
