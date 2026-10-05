import { useText } from "@pengi/shared";
import { Badge } from "@pengi/ui";
import type { ExamOrderStatus } from "@/api/exam-order-service";
import { EXAM_ORDER_STATUS_KEYS } from "@/lib/exam-orders";
import { cn } from "@/lib/utils";

const STATUS_CLASS: Record<ExamOrderStatus, string> = {
	issued: "border-sky-600/40 text-sky-700 dark:text-sky-400",
	partial_results: "border-amber-600/40 text-amber-700 dark:text-amber-400",
	complete_results: "border-green-600/40 text-green-700 dark:text-green-400",
	voided: "border-muted-foreground/40 text-muted-foreground line-through",
};

export function ExamOrderStatusBadge({
	status,
	className,
}: {
	status: ExamOrderStatus;
	className?: string;
}) {
	const { textGet } = useText();
	return (
		<Badge
			variant="outline"
			className={cn(STATUS_CLASS[status] ?? "", className)}
		>
			{textGet(EXAM_ORDER_STATUS_KEYS[status] ?? status)}
		</Badge>
	);
}

/** Small badge for orders with results nobody reviewed yet. */
export function PendingReviewBadge({ className }: { className?: string }) {
	const { textGet } = useText();
	return (
		<Badge variant="secondary" className={className}>
			{textGet("clinical.exam_orders.pending_review")}
		</Badge>
	);
}
