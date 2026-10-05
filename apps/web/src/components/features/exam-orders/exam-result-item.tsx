import { parseDateOnly, useText } from "@pengi/shared";
import { Badge, Button, Checkbox } from "@pengi/ui";
import { Download, Eye, FileText, FileUp, Trash2 } from "lucide-react";
import type { ExamOrderItem } from "@/api/exam-order-service";
import type { PatientAttachment } from "@/api/patient-attachment-service";
import { EXAM_CATEGORY_KEYS, itemHasResult } from "@/lib/exam-orders";

/** Reviewed by whom and when, pending review, or no result yet. */
function ItemReviewBadge({ item }: { item: ExamOrderItem }) {
	const { textGet, formatDateTime } = useText();
	if (item.reviewed_at) {
		const date = formatDateTime(item.reviewed_at);
		return (
			<Badge
				variant="outline"
				className="border-green-600/40 text-green-700 dark:text-green-400"
				title={date}
			>
				{item.reviewed_by_name
					? textGet("clinical.exam_orders.results.reviewed_by", {
							name: item.reviewed_by_name,
							date,
						})
					: textGet("clinical.exam_orders.results.reviewed_at", { date })}
			</Badge>
		);
	}
	if (itemHasResult(item)) {
		return (
			<Badge variant="secondary">
				{textGet("clinical.exam_orders.pending_review")}
			</Badge>
		);
	}
	return (
		<Badge variant="outline" className="text-muted-foreground">
			{textGet("clinical.exam_orders.results.none")}
		</Badge>
	);
}

interface ResultFileRowProps {
	attachment: PatientAttachment;
	canView: boolean;
	canDelete: boolean;
	onView: (attachment: PatientAttachment) => void;
	onDownload: (attachment: PatientAttachment) => void;
	onDelete: (attachment: PatientAttachment) => void;
}

/** One result file: name, exam date, view/download (with access) and delete. */
function ResultFileRow({
	attachment,
	canView,
	canDelete,
	onView,
	onDownload,
	onDelete,
}: ResultFileRowProps) {
	const { textGet, formatDate } = useText();
	return (
		<li className="flex items-center gap-2 text-sm">
			<FileText className="h-4 w-4 shrink-0 text-muted-foreground" />
			<span className="min-w-0 flex-1 truncate">{attachment.file_name}</span>
			<span className="shrink-0 text-xs text-muted-foreground">
				{formatDate(parseDateOnly(attachment.taken_at))}
			</span>
			{canView && (
				<>
					<Button
						type="button"
						variant="ghost"
						size="icon"
						aria-label={textGet("clinical.attachment.view")}
						onClick={() => onView(attachment)}
					>
						<Eye className="h-4 w-4" />
					</Button>
					<Button
						type="button"
						variant="ghost"
						size="icon"
						aria-label={textGet("clinical.attachment.download")}
						onClick={() => onDownload(attachment)}
					>
						<Download className="h-4 w-4" />
					</Button>
				</>
			)}
			{canDelete && (
				<Button
					type="button"
					variant="ghost"
					size="icon"
					aria-label={textGet("clinical.attachment.delete")}
					onClick={() => onDelete(attachment)}
				>
					<Trash2 className="h-4 w-4" />
				</Button>
			)}
		</li>
	);
}

interface ExamResultItemProps {
	item: ExamOrderItem;
	/** Show the review checkbox (REVIEW_EXAM_RESULTS). */
	canReview: boolean;
	selected: boolean;
	onSelect: (checked: boolean) => void;
	/** Show the per-exam upload button. */
	canUpload: boolean;
	onUpload: () => void;
	canViewFiles: boolean;
	/** Whether a given file may be deleted (permission, voided, reviewed). */
	canDeleteFile: (attachment: PatientAttachment) => boolean;
	onView: (attachment: PatientAttachment) => void;
	onDownload: (attachment: PatientAttachment) => void;
	onDelete: (attachment: PatientAttachment) => void;
}

/** One exam of the order with its review state and its result files. */
export function ExamResultItem({
	item,
	canReview,
	selected,
	onSelect,
	canUpload,
	onUpload,
	canViewFiles,
	canDeleteFile,
	onView,
	onDownload,
	onDelete,
}: ExamResultItemProps) {
	const { textGet } = useText();
	const attachments = item.attachments ?? [];
	const selectable = itemHasResult(item) && !item.reviewed_at;
	return (
		<li className="space-y-2 p-3">
			<div className="flex items-start gap-3">
				{canReview && (
					<Checkbox
						className="mt-0.5"
						disabled={!selectable}
						checked={selected}
						onCheckedChange={(checked) => onSelect(checked === true)}
						aria-label={textGet("clinical.exam_orders.results.select_review", {
							name: item.name,
						})}
					/>
				)}
				<div className="min-w-0 flex-1">
					<div className="flex flex-wrap items-center gap-2">
						<span className="text-sm font-medium">{item.name}</span>
						<Badge variant="secondary">
							{textGet(EXAM_CATEGORY_KEYS[item.category])}
						</Badge>
						{item.subgroup && <Badge variant="outline">{item.subgroup}</Badge>}
						<ItemReviewBadge item={item} />
					</div>
					{item.indications && (
						<p className="mt-1 text-xs text-muted-foreground">
							{item.indications}
						</p>
					)}
				</div>
				{canUpload && (
					<Button
						type="button"
						variant="ghost"
						size="icon"
						aria-label={textGet("clinical.exam_orders.results.upload_for", {
							name: item.name,
						})}
						onClick={onUpload}
					>
						<FileUp className="h-4 w-4" />
					</Button>
				)}
			</div>
			{attachments.length > 0 && (
				<ul className="space-y-1 pl-7">
					{attachments.map((attachment) => (
						<ResultFileRow
							key={attachment.ID}
							attachment={attachment}
							canView={canViewFiles}
							canDelete={canDeleteFile(attachment)}
							onView={onView}
							onDownload={onDownload}
							onDelete={onDelete}
						/>
					))}
				</ul>
			)}
		</li>
	);
}
