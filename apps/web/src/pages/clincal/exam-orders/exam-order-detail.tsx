import { useText } from "@pengi/shared";
import { Button, Spinner } from "@pengi/ui";
import { ArrowLeft } from "lucide-react";
import React from "react";
import { Link, useNavigate, useParams } from "react-router";
import { type ExamOrder, getExamOrder } from "@/api/exam-order-service";
import { PageHeader } from "@/components/custom/page-header";
import { ExamOrderActions } from "@/components/features/exam-orders/exam-order-actions";
import {
	ExamOrderInfo,
	VoidedNotice,
} from "@/components/features/exam-orders/exam-order-info";
import { ExamOrderResults } from "@/components/features/exam-orders/exam-order-results";
import {
	ExamOrderStatusBadge,
	PendingReviewBadge,
} from "@/components/features/exam-orders/exam-order-status-badge";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { patientDisplayName } from "@/lib/exam-orders";

/** `/clinical/exam-orders/:id`: an order, its actions and its results. */
export default function ExamOrderDetailPage() {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const orderId = Number(id);
	const { checkPermission } = usePermission();
	const P = PERMISSIONS.EXAM_ORDERS;
	const canUpload = checkPermission([P.PERMISSION_UPLOAD_EXAM_RESULTS]);

	const [order, setOrder] = React.useState<ExamOrder | null>(null);
	const [loading, setLoading] = React.useState(true);

	React.useEffect(() => {
		if (!orderId) return;
		let cancelled = false;
		getExamOrder(orderId).then((res) => {
			if (cancelled) return;
			setOrder(res.success ? (res.data ?? null) : null);
			setLoading(false);
		});
		return () => {
			cancelled = true;
		};
	}, [orderId]);

	if (loading) {
		return (
			<div className="flex h-64 items-center justify-center">
				<Spinner />
			</div>
		);
	}

	if (!order) {
		return (
			<div className="flex h-64 flex-col items-center justify-center gap-4">
				<p className="text-muted-foreground">
					{textGet("clinical.exam_orders.not_found")}
				</p>
				<Button
					variant="outline"
					onClick={() => navigate("/clinical/exam-orders")}
				>
					<ArrowLeft className="mr-2 h-4 w-4" />
					{textGet("clinical.exam_orders.back")}
				</Button>
			</div>
		);
	}

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<OrderHeader order={order} onBack={() => navigate(-1)} />
			<ExamOrderActions
				order={order}
				onChange={setOrder}
				canManage={checkPermission([P.PERMISSION_CREATE_EXAM_ORDER])}
			/>
			<VoidedNotice order={order} />
			<ExamOrderInfo order={order} />
			<ExamOrderResults
				order={order}
				onChange={setOrder}
				canUpload={canUpload}
				canDelete={
					canUpload &&
					checkPermission([
						PERMISSIONS.MEDICAL_RECORD.PERMISSION_DELETE_PATIENT_ATTACHMENT,
					])
				}
				canReview={checkPermission([P.PERMISSION_REVIEW_EXAM_RESULTS])}
				canViewFiles={checkPermission([
					PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_PATIENT_ATTACHMENT,
				])}
			/>
		</main>
	);
}

function OrderHeader({
	order,
	onBack,
}: {
	order: ExamOrder;
	onBack: () => void;
}) {
	const { textGet } = useText();
	const patientName = patientDisplayName(order.patient);
	return (
		<PageHeader
			title={
				<span className="flex flex-wrap items-center gap-2">
					{textGet("clinical.exam_orders.detail.title", { code: order.code })}
					<ExamOrderStatusBadge status={order.status} />
					{order.pending_review && <PendingReviewBadge />}
				</span>
			}
			description={
				patientName ? (
					<Link
						className="hover:underline"
						to={`/clinical/medical-records/${order.patient_id}`}
					>
						{textGet("clinical.exam_orders.patient", { name: patientName })}
					</Link>
				) : undefined
			}
			actions={
				<Button variant="outline" onClick={onBack}>
					<ArrowLeft className="mr-2 h-4 w-4" />
					{textGet("clinical.exam_orders.back")}
				</Button>
			}
		/>
	);
}
