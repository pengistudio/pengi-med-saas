import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Spinner,
} from "@pengi/ui";
import { ChevronRight, FlaskConical, Plus } from "lucide-react";
import React from "react";
import { Link, useNavigate } from "react-router";
import { type ExamOrder, getExamOrders } from "@/api/exam-order-service";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import {
	ExamOrderStatusBadge,
	PendingReviewBadge,
} from "./exam-order-status-badge";

interface ExamOrdersSectionProps {
	patientId: number;
	/** Scope to one consultation; new orders are linked to it. */
	medicalRecordId?: number;
}

/**
 * "Órdenes de exámenes" of a consultation or a patient: their orders with
 * status and pending review, and "Nueva orden" prefilled with the patient
 * (and consultation). Renders nothing without READ_EXAM_ORDER.
 */
export function ExamOrdersSection({
	patientId,
	medicalRecordId,
}: ExamOrdersSectionProps) {
	const { textGet, formatDate } = useText();
	const navigate = useNavigate();
	const { checkPermission } = usePermission();
	const canRead = checkPermission([
		PERMISSIONS.EXAM_ORDERS.PERMISSION_READ_EXAM_ORDER,
	]);
	const canCreate = checkPermission([
		PERMISSIONS.EXAM_ORDERS.PERMISSION_CREATE_EXAM_ORDER,
	]);
	const [orders, setOrders] = React.useState<ExamOrder[]>([]);
	const [loading, setLoading] = React.useState(canRead);

	React.useEffect(() => {
		if (!canRead || !patientId) return;
		let cancelled = false;
		getExamOrders(
			medicalRecordId
				? { record_id: medicalRecordId, limit: 100 }
				: { patient_id: patientId, limit: 100 },
		).then((res) => {
			if (cancelled) return;
			setOrders(res.success ? (res.data?.items ?? []) : []);
			setLoading(false);
		});
		return () => {
			cancelled = true;
		};
	}, [canRead, patientId, medicalRecordId]);

	if (!canRead) return null;

	const newOrderUrl = `/clinical/exam-orders/new?patient_id=${patientId}${
		medicalRecordId ? `&medical_record_id=${medicalRecordId}` : ""
	}`;

	return (
		<Card>
			<CardHeader className="flex flex-row items-start justify-between gap-3">
				<div className="space-y-1.5">
					<CardTitle className="flex items-center gap-2">
						<FlaskConical className="h-5 w-5" />
						{textGet("clinical.exam_orders.title")}
					</CardTitle>
					<CardDescription>
						{textGet("clinical.exam_orders.section.description")}
					</CardDescription>
				</div>
				{canCreate && (
					<Button size="sm" onClick={() => navigate(newOrderUrl)}>
						<Plus className="mr-1.5 h-4 w-4" />
						{textGet("clinical.exam_orders.new")}
					</Button>
				)}
			</CardHeader>
			<CardContent>
				{loading ? (
					<div className="flex justify-center py-6">
						<Spinner />
					</div>
				) : orders.length === 0 ? (
					<p className="py-4 text-center text-sm text-muted-foreground">
						{textGet("clinical.exam_orders.empty")}
					</p>
				) : (
					<ul className="divide-y rounded-md border">
						{orders.map((order) => (
							<li key={order.ID}>
								<Link
									to={`/clinical/exam-orders/${order.ID}`}
									className="flex items-center gap-3 p-3 text-sm hover:bg-muted/50"
								>
									<div className="min-w-0 flex-1 space-y-1">
										<div className="flex flex-wrap items-center gap-2">
											<span className="font-medium">{order.code}</span>
											<ExamOrderStatusBadge status={order.status} />
											{order.pending_review && <PendingReviewBadge />}
										</div>
										<p className="text-xs text-muted-foreground">
											{formatDate(order.CreatedAt)} ·{" "}
											{textGet("clinical.exam_orders.exam_count", {
												count: order.items?.length ?? 0,
											})}
										</p>
									</div>
									<ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
								</Link>
							</li>
						))}
					</ul>
				)}
			</CardContent>
		</Card>
	);
}
