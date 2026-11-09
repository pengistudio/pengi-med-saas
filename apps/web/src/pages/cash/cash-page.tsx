import { useText } from "@pengi/shared";
import { Button, Skeleton } from "@pengi/ui";
import {
	ArrowLeftRight,
	CircleAlert,
	Clock,
	History,
	Lock,
	RotateCw,
	TriangleAlert,
	Wallet,
} from "lucide-react";
import { type ReactNode, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import {
	type CashSession,
	type CashSessionDetail,
	getSessions,
	type MovementType,
	openSession,
	type PaymentView,
} from "@/api/cash-service";
import { PageHeader } from "@/components/custom/page-header";
import { CloseSessionDialog } from "@/components/features/cash/close-session-dialog";
import {
	DeskPanel,
	LastCloseCard,
} from "@/components/features/cash/desk-panel";
import { MovementDialog } from "@/components/features/cash/movement-dialog";
import { SessionActivity } from "@/components/features/cash/session-activity";
import { VoidPaymentDialog } from "@/components/features/cash/void-payment-dialog";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { getMovementColumns } from "@/sections/columns/cash/movement-columns";
import { getPaymentColumns } from "@/sections/columns/cash/payment-columns";
import {
	OpenSessionForm,
	type OpenSessionValues,
} from "@/sections/forms/cash/open-session-form";
import { useCashStore } from "@/store/cash-store";

/**
 * The cash desk. Closed: opening it is the page's main action, with the last
 * close underneath as a compact block. Open: the desk's state (who, since
 * when, over 24 h) with its actions, then the session's payments and
 * movements. The open session never shows the opening float, cash totals or
 * expected cash (blind close).
 */
const CashPage = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { checkPermission } = usePermission();
	const session = useCashStore((s) => s.session);
	const loaded = useCashStore((s) => s.loaded);
	const loadError = useCashStore((s) => s.error);
	const fetchCurrent = useCashStore((s) => s.fetchCurrent);
	const setSession = useCashStore((s) => s.setSession);
	const [lastClose, setLastClose] = useState<CashSession | null>(null);
	const [opening, setOpening] = useState(false);
	const [closeOpen, setCloseOpen] = useState(false);
	const [movementOpen, setMovementOpen] = useState(false);
	const [voiding, setVoiding] = useState<PaymentView | null>(null);

	const canManage = checkPermission([
		PERMISSIONS.CASH.PERMISSION_MANAGE_CASH_SESSION,
	]);
	const canVoid = checkPermission([PERMISSIONS.CASH.PERMISSION_VOID_PAYMENT]);
	const movementTypes = useMemo<MovementType[]>(
		() => [
			...(canManage ? (["in"] as const) : []),
			...(canVoid ? (["out"] as const) : []),
		],
		[canManage, canVoid],
	);

	useEffect(() => {
		fetchCurrent();
	}, [fetchCurrent]);

	// With the desk closed, the most recent close (newest first).
	const deskClosed = loaded && !loadError && !session;
	useEffect(() => {
		if (!deskClosed) return;
		let cancelled = false;
		getSessions({ page: 1, limit: 1, status: "closed" }).then((res) => {
			if (!cancelled && res.success) setLastClose(res.data?.items[0] ?? null);
		});
		return () => {
			cancelled = true;
		};
	}, [deskClosed]);

	const paymentColumns = useMemo(
		() => getPaymentColumns({ onVoid: canVoid ? setVoiding : undefined }),
		[canVoid],
	);
	const movementColumns = useMemo(() => getMovementColumns(), []);

	async function handleOpen(values: OpenSessionValues) {
		setOpening(true);
		const res = await openSession(values.opening_float);
		setOpening(false);
		if (res.success) setSession(res.data);
	}

	function handleClosed(detail: CashSessionDetail) {
		setLastClose(detail);
		setSession(null);
	}

	return (
		<main className="grid grid-cols-1 items-start gap-6">
			<PageHeader
				title={textGet("cash.page.title")}
				description={textGet("cash.page.description")}
				actions={
					<Button variant="outline" onClick={() => navigate("/cash/history")}>
						<History className="h-4 w-4" />
						{textGet("cash.history.title")}
					</Button>
				}
			/>

			{!loaded ? (
				<CashPageSkeleton />
			) : loadError && !session ? (
				<LoadError onRetry={() => fetchCurrent()} />
			) : session ? (
				<>
					<OpenDesk
						session={session}
						actions={
							<>
								{movementTypes.length > 0 && (
									<Button
										variant="outline"
										size="lg"
										onClick={() => setMovementOpen(true)}
									>
										<ArrowLeftRight className="h-4 w-4" />
										{textGet("cash.movement.button")}
									</Button>
								)}
								{canManage && (
									<Button size="lg" onClick={() => setCloseOpen(true)}>
										<Lock className="h-4 w-4" />
										{textGet("cash.close.button")}
									</Button>
								)}
							</>
						}
					/>
					<SessionActivity
						payments={session.payments}
						movements={session.movements}
						paymentColumns={paymentColumns}
						movementColumns={movementColumns}
						paymentsHint
						movementsHint={movementTypes.length > 0}
					/>
				</>
			) : (
				<>
					<DeskPanel
						tone="closed"
						icon={Wallet}
						title={textGet("cash.desk.closed.title")}
						meta={
							canManage
								? textGet("cash.open.description")
								: textGet("cash.closed.no_permission")
						}
					>
						{canManage && (
							<OpenSessionForm loading={opening} onSubmit={handleOpen} />
						)}
					</DeskPanel>
					{lastClose && <LastCloseCard session={lastClose} />}
				</>
			)}

			{session && (
				<>
					<CloseSessionDialog
						sessionId={session.ID}
						open={closeOpen}
						onOpenChange={setCloseOpen}
						onClosed={handleClosed}
					/>
					<MovementDialog
						open={movementOpen}
						allowedTypes={movementTypes}
						onOpenChange={setMovementOpen}
						onCreated={() => fetchCurrent()}
					/>
				</>
			)}
			<VoidPaymentDialog
				payment={voiding}
				onOpenChange={(open) => !open && setVoiding(null)}
				onVoided={() => fetchCurrent()}
			/>
		</main>
	);
};

/** The open desk's state: who opened it, since when, and a 24 h warning. */
function OpenDesk({
	session,
	actions,
}: {
	session: CashSessionDetail;
	actions: ReactNode;
}) {
	const { textGet, formatDateTime, formatRelative } = useText();

	return (
		<DeskPanel
			tone={session.stale ? "stale" : "open"}
			icon={Wallet}
			title={textGet("cash.session.open.title")}
			meta={
				<div className="flex flex-wrap items-center gap-x-3 gap-y-1">
					<span>
						{textGet("cash.session.open.since", {
							name: session.opened_by_name || "—",
							date: formatDateTime(session.opened_at),
						})}
					</span>
					<span
						className={
							session.stale
								? "inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-xs font-medium text-amber-700 dark:text-amber-400"
								: "inline-flex items-center gap-1 text-xs"
						}
					>
						<Clock aria-hidden className="h-3.5 w-3.5" />
						{session.stale
							? textGet("cash.session.stale.badge")
							: formatRelative(session.opened_at)}
					</span>
				</div>
			}
			actions={actions}
		>
			{session.stale && (
				<p
					role="alert"
					className="flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-800 dark:text-amber-300"
				>
					<TriangleAlert aria-hidden className="mt-0.5 h-4 w-4 shrink-0" />
					{textGet("cash.session.stale")}
				</p>
			)}
		</DeskPanel>
	);
}

/** The shape of the desk panel and the activity while the desk loads. */
function CashPageSkeleton() {
	return (
		<div className="space-y-6" aria-hidden>
			<div className="rounded-xl p-4 ring-1 ring-foreground/10 sm:p-6">
				<div className="flex items-start gap-3">
					<Skeleton className="h-10 w-10 rounded-full" />
					<div className="flex-1 space-y-2">
						<Skeleton className="h-6 w-48" />
						<Skeleton className="h-4 w-72 max-w-full" />
					</div>
				</div>
			</div>
			<div className="space-y-3">
				<Skeleton className="h-8 w-56" />
				<Skeleton className="h-40 w-full" />
			</div>
		</div>
	);
}

function LoadError({ onRetry }: { onRetry: () => void }) {
	const { textGet } = useText();
	return (
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
			<Button variant="outline" onClick={onRetry}>
				<RotateCw className="h-4 w-4" />
				{textGet("cash.error.retry")}
			</Button>
		</div>
	);
}

export default CashPage;
