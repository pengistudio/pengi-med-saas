import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";

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
	if (!prescription) {
		return null;
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[425px]">
				<DialogHeader>
					<DialogTitle>Receta Médica</DialogTitle>
					<DialogDescription>
						Detalles de la última prescripción emitida para este paciente.
					</DialogDescription>
				</DialogHeader>
				<div className="grid gap-4 py-4">
					<div className="grid gap-2">
						<h4 className="font-medium text-sm">Contenido</h4>
						<p className="text-sm bg-muted p-3 rounded-md whitespace-pre-wrap">
							{prescription.content}
						</p>
					</div>
					<div className="grid gap-2">
						<h4 className="font-medium text-sm">Indicaciones</h4>
						<p className="text-sm bg-muted p-3 rounded-md whitespace-pre-wrap">
							{prescription.indications}
						</p>
					</div>
				</div>
			</DialogContent>
		</Dialog>
	);
}
