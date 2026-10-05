import { useText } from "@pengi/shared";
import { Badge, Card, CardContent, CardHeader, CardTitle } from "@pengi/ui";
import type React from "react";
import { Link } from "react-router";
import type { ExamOrder } from "@/api/exam-order-service";
import { EXAM_PRIORITY_KEYS, isOrderVoided } from "@/lib/exam-orders";

function DetailRow({
	label,
	children,
	className,
}: {
	label: string;
	children: React.ReactNode;
	className?: string;
}) {
	return (
		<div className={className}>
			<dt className="text-muted-foreground">{label}</dt>
			<dd className="mt-0.5">{children}</dd>
		</div>
	);
}

/** When and why a voided order was voided; nothing for other orders. */
export function VoidedNotice({ order }: { order: ExamOrder }) {
	const { textGet, formatDateTime } = useText();
	if (!isOrderVoided(order)) return null;
	return (
		<div
			role="status"
			className="rounded-md border border-destructive/40 bg-destructive/5 p-3 text-sm"
		>
			<p className="font-medium text-destructive">
				{textGet("clinical.exam_orders.voided.title", {
					date: formatDateTime(order.voided_at),
				})}
			</p>
			{order.void_reason && (
				<p className="mt-1 whitespace-pre-wrap">{order.void_reason}</p>
			)}
		</div>
	);
}

/** The order's header data: who and when, priority, lab, diagnoses, notes. */
export function ExamOrderInfo({ order }: { order: ExamOrder }) {
	const { textGet, formatDateTime } = useText();
	const diagnoses = order.diagnoses ?? [];
	return (
		<Card>
			<CardHeader>
				<CardTitle>{textGet("clinical.exam_orders.form.details")}</CardTitle>
			</CardHeader>
			<CardContent>
				<dl className="grid gap-4 text-sm sm:grid-cols-2 lg:grid-cols-3">
					<DetailRow label={textGet("clinical.exam_orders.field.ordered_by")}>
						{order.ordered_by_name || "—"}
					</DetailRow>
					<DetailRow label={textGet("clinical.exam_orders.field.date")}>
						{formatDateTime(order.CreatedAt)}
					</DetailRow>
					<DetailRow label={textGet("clinical.exam_orders.field.priority")}>
						{order.priority === "urgent" ? (
							<Badge variant="destructive">
								{textGet(EXAM_PRIORITY_KEYS.urgent)}
							</Badge>
						) : (
							textGet(EXAM_PRIORITY_KEYS[order.priority] ?? order.priority)
						)}
					</DetailRow>
					<DetailRow
						label={textGet("clinical.exam_orders.field.destination_lab")}
					>
						{order.destination_lab || "—"}
					</DetailRow>
					<DetailRow label={textGet("clinical.exam_orders.field.diagnoses")}>
						{diagnoses.length > 0
							? diagnoses.map((d) => `${d.code} ${d.title}`).join("; ")
							: "—"}
					</DetailRow>
					{order.medical_record_id && (
						<DetailRow label={textGet("clinical.exam_orders.field.record")}>
							<Link
								className="text-primary underline-offset-4 hover:underline"
								to={`/clinical/medical-records/view/${order.medical_record_id}`}
							>
								{textGet("clinical.exam_orders.field.record.open")}
							</Link>
						</DetailRow>
					)}
					{order.notes && (
						<DetailRow
							label={textGet("clinical.exam_orders.field.notes")}
							className="sm:col-span-2 lg:col-span-3"
						>
							<span className="whitespace-pre-wrap">{order.notes}</span>
						</DetailRow>
					)}
				</dl>
			</CardContent>
		</Card>
	);
}
