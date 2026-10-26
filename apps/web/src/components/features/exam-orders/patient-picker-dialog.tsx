import { useText } from "@pengi/shared";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";
import type { Patient } from "@/api/clinical-service";
import { PatientSearch } from "@/components/features/patients/patient-search";

interface PatientPickerDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onPick: (patient: Patient) => void;
}

/** Searches a patient by name or document and hands back the chosen one. */
export function PatientPickerDialog({
	open,
	onOpenChange,
	onPick,
}: PatientPickerDialogProps) {
	const { textGet } = useText();
	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>
						{textGet("clinical.exam_orders.patient_picker.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet("clinical.exam_orders.patient_picker.description")}
					</DialogDescription>
				</DialogHeader>
				<PatientSearch active={open} onPick={onPick} />
			</DialogContent>
		</Dialog>
	);
}
