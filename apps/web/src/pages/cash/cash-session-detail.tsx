import { useText } from "@pengi/shared";
import { Button, Skeleton } from "@pengi/ui";
import { ArrowLeft, CircleAlert, RotateCw } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router";
import {
	type CashSessionDetail,
	getSession,
	type PaymentView,
} from "@/api/cash-service";
import { PageHeader } from "@/components/custom/page-header";
import { CashEmptyState } from "@/components/features/cash/cash-figures";
import { RefundPaymentDialog } from "@/components/features/cash/refund-payment-dialog";
import { SessionActivity } from "@/components/features/cash/session-activity";
import { SessionCloseSummary } from "@/components/features/cash/session-close-summary";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { getMovementColumns } from "@/sections/columns/cash/movement-columns";
import { getPaymentColumns } from "@/sections/columns/cash/payment-columns";
import { useCashStore } from "@/store/cash-store";

type LoadState = "loading" | "ready" | "not_found" | "error";

/**
 * A closed session: totals, difference, payments, movements and its closing
 * report. Its payments can be refunded into the open session. The open
 * session lives on /cash (blind close), so it redirects there.
 */
const CashSessionDetailPage = () => {
	const { textGet, formatDate } = useText();
	const navigate = useNavigate();
	const { id } = useParams();
	const sessionId = Number(id);
	const { checkPermission } = usePermission();
	const currentSession = useCashStore((s) => s.session);
	const fetchCurrent = useCashStore((s) => s.fetchCurrent);
	const [detail, setDetail] = useState<CashSessionDetail | null>(null);
	const [state, setState] = useState<LoadState>("loading");
	const [refunding, setRefunding] = useState<PaymentView | null>(null);

	const load = useCallback(async () => {
		setState("loading");
		const res = await getSession(sessionId);
		setDetail(res.success ? res.data : null);
		setState(res.success ? "ready" : res.code === 404 ? "not_found" : "error");
	}, [sessionId]);

	useEffect(() => {
		load();
		fetchCurrent();
	}, [load, fetchCurrent]);

	// Refunds go into the open session: only offered while one is open.
	const canRefund =
		checkPermission([PERMISSIONS.CASH.PERMISSION_VOID_PAYMENT]) &&
		Boolean(currentSession);
	const paymentColumns = useMemo(
		() => getPaymentColumns({ onRefund: canRefund ? setRefunding : undefined }),
		[canRefund],
	);
	const movementColumns = useMemo(() => getMovementColumns(), []);

	if (detail?.status === "open") return <Navigate to="/cash" replace />;

	return (
		<main className="grid grid-cols-1 items-start gap-6">
			<PageHeader
				title={
					detail ? (
						textGet("cash.detail.title", {
							date: formatDate(detail.opened_at, "long"),
						})
					) : state === "loading" ? (
						<Skeleton className="h-8 w-64 max-w-full" />
					) : (
						textGet("cash.history.title")
					)
				}
				actions={
					<Button variant="outline" onClick={() => navigate("/cash/history")}>
						<ArrowLeft className="h-4 w-4" />
						{textGet("cash.history.title")}
					</Button>
				}
			/>

			{state === "loading" ? (
				<div className="space-y-6" aria-hidden>
					<Skeleton className="h-72 w-full rounded-xl" />
					<Skeleton className="h-8 w-56" />
					<Skeleton className="h-40 w-full" />
				</div>
			) : state === "not_found" || (state === "ready" && !detail) ? (
				<div className="rounded-xl border border-dashed">
					<CashEmptyState
						illustration="not-found"
						title={textGet("cash.detail.not_found")}
					/>
				</div>
			) : state === "error" || !detail ? (
				<div
					role="alert"
					className="flex flex-col items-start gap-3 rounded-xl border border-destructive/30 bg-destructive/5 p-4 sm:flex-row sm:items-center sm:justify-between"
				>
					<p className="flex items-start gap-2 text-sm">
						<CircleAlert
							aria-hidden
							className="mt-0.5 h-4 w-4 shrink-0 text-destructive"
						/>
						{textGet("cash.error.load")}
					</p>
					<Button variant="outline" onClick={load}>
						<RotateCw className="h-4 w-4" />
						{textGet("cash.error.retry")}
					</Button>
				</div>
			) : (
				<>
					<SessionCloseSummary session={detail} />
					<SessionActivity
						payments={detail.payments}
						movements={detail.movements}
						paymentColumns={paymentColumns}
						movementColumns={movementColumns}
					/>
				</>
			)}

			<RefundPaymentDialog
				payment={refunding}
				onOpenChange={(open) => !open && setRefunding(null)}
				onRefunded={() => fetchCurrent()}
			/>
		</main>
	);
};

export default CashSessionDetailPage;
