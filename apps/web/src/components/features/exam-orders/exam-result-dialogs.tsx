import { useText } from "@pengi/shared";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";
import type { ExamOrder } from "@/api/exam-order-service";
import type { PatientAttachment } from "@/api/patient-attachment-service";
import { saveAttachmentFile } from "@/components/features/patient-attachments/attachment-actions";
import { AttachmentViewer } from "@/components/features/patient-attachments/attachment-viewer";
import { itemsOfAttachment, orderAttachments } from "@/lib/exam-orders";
import { PatientAttachmentDeleteForm } from "@/sections/forms/clinical/patient-attachment-form";

/** The attachments viewer stepping through every result file of the order. */
export function OrderFilesViewer({
	order,
	viewingId,
	onView,
}: {
	order: ExamOrder;
	/** The file shown; null keeps the viewer closed. */
	viewingId: number | null;
	onView: (attachmentId: number | null) => void;
}) {
	const { textGet } = useText();
	const files = orderAttachments(order);
	const index = files.findIndex((f) => f.ID === viewingId);
	const viewing = index >= 0 ? files[index] : null;
	const covers = viewing
		? itemsOfAttachment(order, viewing.ID)
				.map((item) => item.name)
				.join(", ")
		: "";
	return (
		<AttachmentViewer
			patientId={order.patient_id}
			attachment={viewing}
			onClose={() => onView(null)}
			onDownload={(attachment) =>
				saveAttachmentFile(order.patient_id, attachment)
			}
			onPrevious={index > 0 ? () => onView(files[index - 1].ID) : undefined}
			onNext={
				index >= 0 && index < files.length - 1
					? () => onView(files[index + 1].ID)
					: undefined
			}
			position={
				files.length > 1 && index >= 0
					? `${index + 1} / ${files.length}`
					: undefined
			}
			details={
				viewing ? (
					<p className="text-xs text-muted-foreground">
						{textGet("clinical.exam_orders.results.covers_list", {
							names: covers,
						})}
					</p>
				) : null
			}
		/>
	);
}

/** Asks the reason before a result file is deleted. */
export function DeleteResultDialog({
	attachment,
	busy,
	onCancel,
	onConfirm,
}: {
	/** The file to delete; null keeps the dialog closed. */
	attachment: PatientAttachment | null;
	busy: boolean;
	onCancel: () => void;
	onConfirm: (reason: string) => void;
}) {
	const { textGet } = useText();
	return (
		<Dialog
			open={attachment !== null}
			onOpenChange={(open) => {
				if (!open && !busy) onCancel();
			}}
		>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>
						{textGet("clinical.exam_orders.results.delete.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet("clinical.exam_orders.results.delete.description", {
							name: attachment?.file_name ?? "",
						})}
					</DialogDescription>
				</DialogHeader>
				<PatientAttachmentDeleteForm
					loading={busy}
					onSubmit={(values) => onConfirm(values.reason)}
				/>
			</DialogContent>
		</Dialog>
	);
}
