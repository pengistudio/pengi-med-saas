import { useText } from "@pengi/shared";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";

export interface Prescription {
	content: string;
	indications: string;
}

interface PrescriptionDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	prescription?: Prescription | null;
}

export default function PrescriptionDialog({
	open,
	onOpenChange,
	prescription,
}: PrescriptionDialogProps) {
	const { textGet } = useText();
	if (!prescription) {
		return null;
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[425px]">
				<DialogHeader>
					<DialogTitle>{textGet("dialog.prescription.title")}</DialogTitle>
					<DialogDescription>
						{textGet("dialog.prescription.description")}
					</DialogDescription>
				</DialogHeader>
				<div className="grid gap-4 py-4">
					<div className="grid gap-2">
						<h4 className="font-medium text-sm">
							{textGet("dialog.prescription.content")}
						</h4>
						<p className="text-sm bg-muted p-3 rounded-md whitespace-pre-wrap">
							{prescription.content}
						</p>
					</div>
					<div className="grid gap-2">
						<h4 className="font-medium text-sm">
							{textGet("dialog.prescription.indications")}
						</h4>
						<p className="text-sm bg-muted p-3 rounded-md whitespace-pre-wrap">
							{prescription.indications}
						</p>
					</div>
				</div>
			</DialogContent>
		</Dialog>
	);
}
