import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@pengi/ui";
import { CheckCheck, FileUp } from "lucide-react";
import React from "react";
import {
	deleteExamResult,
	type ExamOrder,
	reviewExamResults,
} from "@/api/exam-order-service";
import type { PatientAttachment } from "@/api/patient-attachment-service";
import { saveAttachmentFile } from "@/components/features/patient-attachments/attachment-actions";
import {
	isOrderVoided,
	itemHasResult,
	itemsOfAttachment,
} from "@/lib/exam-orders";
import { DeleteResultDialog, OrderFilesViewer } from "./exam-result-dialogs";
import { ExamResultItem } from "./exam-result-item";
import { ExamResultsUploadDialog } from "./exam-results-upload-dialog";

interface ExamOrderResultsProps {
	order: ExamOrder;
	onChange: (order: ExamOrder) => void;
	canUpload: boolean;
	/** Deleting a result also needs DELETE_PATIENT_ATTACHMENT. */
	canDelete: boolean;
	canReview: boolean;
	/** READ_PATIENT_ATTACHMENT: without it only the files' metadata shows. */
	canViewFiles: boolean;
}

/**
 * The results of an order: per exam, the files that cover it (viewer with
 * navigation across the order's files, download, delete with a reason), the
 * upload marking which exams a file covers, and the review of selected exams.
 */
export function ExamOrderResults({
	order,
	onChange,
	canUpload,
	canDelete,
	canReview,
	canViewFiles,
}: ExamOrderResultsProps) {
	const { textGet } = useText();
	const voided = isOrderVoided(order);
	const [uploadOpen, setUploadOpen] = React.useState(false);
	const [uploadItems, setUploadItems] = React.useState<number[] | undefined>();
	const [selected, setSelected] = React.useState<number[]>([]);
	const [reviewing, setReviewing] = React.useState(false);
	const [viewingId, setViewingId] = React.useState<number | null>(null);
	const [deleting, setDeleting] = React.useState<PatientAttachment | null>(
		null,
	);
	const [deletingBusy, setDeletingBusy] = React.useState(false);

	const reviewableIds = new Set(
		order.items
			.filter((item) => itemHasResult(item) && !item.reviewed_at)
			.map((item) => item.ID),
	);
	const selectedReviewable = selected.filter((id) => reviewableIds.has(id));
	const selectedIds = new Set(selected);

	async function handleReview() {
		if (selectedReviewable.length === 0) return;
		setReviewing(true);
		try {
			const res = await reviewExamResults(order.ID, selectedReviewable);
			if (res.success && res.data) {
				setSelected([]);
				onChange(res.data);
			}
		} finally {
			setReviewing(false);
		}
	}

	async function handleDelete(reason: string) {
		if (!deleting) return;
		setDeletingBusy(true);
		try {
			const res = await deleteExamResult(order.ID, deleting.ID, reason.trim());
			if (res.success && res.data) {
				setDeleting(null);
				if (viewingId === deleting.ID) setViewingId(null);
				onChange(res.data);
			}
		} finally {
			setDeletingBusy(false);
		}
	}

	function openUpload(itemIds?: number[]) {
		setUploadItems(itemIds);
		setUploadOpen(true);
	}

	// A reviewed exam's results can't be deleted (the server refuses too).
	const canDeleteFile = (attachment: PatientAttachment) =>
		canDelete &&
		!voided &&
		!itemsOfAttachment(order, attachment.ID).some((i) => i.reviewed_at);

	const toggle = (id: number, checked: boolean) =>
		setSelected((current) =>
			checked ? [...current, id] : current.filter((v) => v !== id),
		);

	return (
		<Card>
			<CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
				<div className="space-y-1.5">
					<CardTitle>{textGet("clinical.exam_orders.results.title")}</CardTitle>
					<CardDescription>
						{textGet("clinical.exam_orders.results.description")}
					</CardDescription>
				</div>
				<div className="flex flex-wrap gap-2">
					{canReview && reviewableIds.size > 0 && (
						<Button
							type="button"
							variant="outline"
							disabled={selectedReviewable.length === 0 || reviewing}
							onClick={handleReview}
						>
							<CheckCheck className="mr-2 h-4 w-4" />
							{textGet("clinical.exam_orders.results.review", {
								count: selectedReviewable.length,
							})}
						</Button>
					)}
					{canUpload && !voided && (
						<Button type="button" onClick={() => openUpload()}>
							<FileUp className="mr-2 h-4 w-4" />
							{textGet("clinical.exam_orders.results.upload")}
						</Button>
					)}
				</div>
			</CardHeader>
			<CardContent>
				<ul className="divide-y rounded-md border">
					{order.items.map((item) => (
						<ExamResultItem
							key={item.ID}
							item={item}
							canReview={canReview}
							selected={selectedIds.has(item.ID)}
							onSelect={(checked) => toggle(item.ID, checked)}
							canUpload={canUpload && !voided}
							onUpload={() => openUpload([item.ID])}
							canViewFiles={canViewFiles}
							canDeleteFile={canDeleteFile}
							onView={(attachment) => setViewingId(attachment.ID)}
							onDownload={(attachment) =>
								saveAttachmentFile(order.patient_id, attachment)
							}
							onDelete={setDeleting}
						/>
					))}
				</ul>
			</CardContent>

			{canUpload && (
				<ExamResultsUploadDialog
					order={order}
					open={uploadOpen}
					initialItemIds={uploadItems}
					onOpenChange={setUploadOpen}
					onUploaded={onChange}
				/>
			)}
			{canViewFiles && (
				<OrderFilesViewer
					order={order}
					viewingId={viewingId}
					onView={setViewingId}
				/>
			)}
			<DeleteResultDialog
				attachment={deleting}
				busy={deletingBusy}
				onCancel={() => setDeleting(null)}
				onConfirm={handleDelete}
			/>
		</Card>
	);
}
