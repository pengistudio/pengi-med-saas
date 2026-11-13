import { useText } from "@pengi/shared";
import { Card, CardContent } from "@pengi/ui";
import { MessageSquareText } from "lucide-react";
import type React from "react";
import type { CashSessionDetail } from "@/api/cash-service";
import { cn } from "@/lib/utils";
import {
	CashEmptyState,
	DifferenceBadge,
	MethodIcon,
	Outflow,
	useSessionRange,
} from "./cash-figures";
import { paymentMethodLabel } from "./cash-labels";
import { SessionReportButton } from "./session-report-button";

/** One line of the cash ledger: label left, figure right (tabular). */
function LedgerRow({
	label,
	children,
	className,
}: {
	label: string;
	children: React.ReactNode;
	className?: string;
}) {
	return (
		<div className={cn("flex items-baseline justify-between gap-4", className)}>
			<dt className="text-muted-foreground">{label}</dt>
			<dd className="text-right tabular-nums">{children}</dd>
		</div>
	);
}

/**
 * The figures of a closed session, read as the drawer's arithmetic: float,
 * plus cash payments and inflows, minus refunds and outflows, equals the
 * expected cash; then the count and the difference (text and colour). Next to
 * it, the totals per method. Only for closed sessions: the open one never
 * shows them (blind close).
 */
export function SessionCloseSummary({
	session,
}: {
	session: CashSessionDetail;
}) {
	const { textGet, formatMoney } = useText();
	const range = useSessionRange();

	return (
		<Card>
			<CardContent className="space-y-6">
				<header className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
					<div className="space-y-1">
						<h2 className="text-lg font-semibold">
							{textGet("cash.summary.title")}
						</h2>
						<p className="text-sm">
							{range(session.opened_at, session.closed_at)}
						</p>
						<p className="text-sm text-muted-foreground">
							{textGet("cash.summary.people", {
								opened_by: session.opened_by_name || "—",
								closed_by: session.closed_by_name || "—",
							})}
						</p>
					</div>
					<SessionReportButton sessionId={session.ID} />
				</header>

				<div className="grid gap-6 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
					<section aria-labelledby="cash-summary-cash" className="space-y-3">
						<h3 id="cash-summary-cash" className="text-sm font-medium">
							{textGet("cash.summary.cash")}
						</h3>
						<dl className="space-y-2 text-sm">
							<LedgerRow label={textGet("cash.summary.opening_float")}>
								{formatMoney(session.opening_float)}
							</LedgerRow>
							<LedgerRow label={textGet("cash.summary.cash_in")}>
								{formatMoney(session.totals.cash_in)}
							</LedgerRow>
							<LedgerRow label={textGet("cash.summary.cash_refunded")}>
								<Outflow amount={session.totals.cash_refunded} />
							</LedgerRow>
							<LedgerRow label={textGet("cash.summary.movements_in")}>
								{formatMoney(session.totals.movements_in)}
							</LedgerRow>
							<LedgerRow label={textGet("cash.summary.movements_out")}>
								<Outflow amount={session.totals.movements_out} />
							</LedgerRow>
							<LedgerRow
								label={textGet("cash.summary.expected")}
								className="border-t pt-2 font-medium [&>dt]:text-foreground"
							>
								{formatMoney(session.expected_cash)}
							</LedgerRow>
							<LedgerRow
								label={textGet("cash.summary.counted")}
								className="font-medium [&>dt]:text-foreground"
							>
								{formatMoney(session.counted_cash)}
							</LedgerRow>
						</dl>
						<div className="flex flex-wrap items-center justify-between gap-2 rounded-lg bg-muted/50 px-3 py-2.5 text-sm">
							<span className="font-medium">
								{textGet("cash.summary.difference")}
							</span>
							<DifferenceBadge value={session.difference ?? 0} />
						</div>
						{session.close_note && (
							<figure className="flex gap-3 rounded-lg border-l-2 border-primary/50 bg-muted/30 py-2.5 pr-3 pl-3 text-sm">
								<MessageSquareText
									aria-hidden
									className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground"
								/>
								<div className="min-w-0 space-y-0.5">
									<figcaption className="text-xs font-medium text-muted-foreground">
										{textGet("cash.summary.note")}
									</figcaption>
									<blockquote className="whitespace-pre-line break-words">
										{session.close_note}
									</blockquote>
								</div>
							</figure>
						)}
					</section>

					<section aria-labelledby="cash-summary-methods" className="space-y-3">
						<h3 id="cash-summary-methods" className="text-sm font-medium">
							{textGet("cash.summary.by_method")}
						</h3>
						{session.totals.by_method.length === 0 ? (
							<div className="rounded-lg border border-dashed">
								<CashEmptyState
									illustration="empty"
									title={textGet("cash.summary.no_payments")}
								/>
							</div>
						) : (
							<ul className="divide-y rounded-lg border">
								{session.totals.by_method.map((m) => (
									<li
										key={m.method}
										className="flex items-center gap-3 px-3 py-2.5 text-sm"
									>
										<span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
											<MethodIcon method={m.method} />
										</span>
										<span className="min-w-0 flex-1 truncate">
											{textGet("cash.summary.method_count", {
												method: paymentMethodLabel(textGet, m.method),
												count: m.count,
											})}
										</span>
										<span className="font-medium tabular-nums">
											{formatMoney(m.total)}
										</span>
									</li>
								))}
							</ul>
						)}
					</section>
				</div>
			</CardContent>
		</Card>
	);
}
