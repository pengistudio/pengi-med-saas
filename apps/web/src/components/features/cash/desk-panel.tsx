import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { Eye, type LucideIcon } from "lucide-react";
import type React from "react";
import { useNavigate } from "react-router";
import type { CashSession } from "@/api/cash-service";
import { cn } from "@/lib/utils";
import { DifferenceBadge, useSessionRange } from "./cash-figures";
import { SessionReportButton } from "./session-report-button";

export type DeskTone = "open" | "stale" | "closed";

const RAIL: Record<DeskTone, string> = {
	open: "before:bg-primary",
	stale: "before:bg-amber-500",
	closed: "before:bg-muted-foreground/30",
};

const BADGE: Record<DeskTone, string> = {
	open: "bg-primary/10 text-primary",
	stale: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
	closed: "bg-muted text-muted-foreground",
};

/**
 * The state of the cash desk, at the top of /cash: a rail coloured by state
 * (open, open for over 24 h, closed), the state as the heading and the
 * action of the moment (open the desk, record a movement, close it).
 */
export function DeskPanel({
	tone,
	icon: Icon,
	title,
	meta,
	actions,
	children,
}: {
	tone: DeskTone;
	icon: LucideIcon;
	title: string;
	/** Who and since when; under the title. */
	meta?: React.ReactNode;
	actions?: React.ReactNode;
	children?: React.ReactNode;
}) {
	return (
		<section
			aria-label={title}
			className={cn(
				"relative overflow-hidden rounded-xl bg-card p-4 text-card-foreground ring-1 ring-foreground/10 sm:p-6",
				"before:absolute before:inset-y-0 before:left-0 before:w-1",
				RAIL[tone],
			)}
		>
			<div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
				<div className="flex min-w-0 items-start gap-3">
					<span
						className={cn(
							"flex h-10 w-10 shrink-0 items-center justify-center rounded-full",
							BADGE[tone],
						)}
					>
						<Icon aria-hidden className="h-5 w-5" />
					</span>
					<div className="min-w-0 space-y-1">
						<h2 className="text-xl font-semibold tracking-tight">{title}</h2>
						{meta && (
							<div className="text-sm text-muted-foreground">{meta}</div>
						)}
					</div>
				</div>
				{actions && (
					<div className="flex flex-wrap gap-2 *:flex-1 sm:shrink-0 sm:*:flex-none">
						{actions}
					</div>
				)}
			</div>
			{children && <div className="mt-4 sm:pl-[3.25rem]">{children}</div>}
		</section>
	);
}

/**
 * The last closed session, compact: when, who closed it, the count and the
 * difference, with its PDF and a link to the full detail.
 */
export function LastCloseCard({ session }: { session: CashSession }) {
	const { textGet, formatMoney } = useText();
	const navigate = useNavigate();
	const range = useSessionRange();

	return (
		<section
			aria-labelledby="cash-last-close"
			className="rounded-xl border border-dashed p-4"
		>
			<div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
				<div className="min-w-0 space-y-1">
					<h2 id="cash-last-close" className="text-sm font-medium">
						{textGet("cash.last_close.title")}
					</h2>
					<p className="text-sm text-muted-foreground">
						{range(session.opened_at, session.closed_at)}
					</p>
					<p className="text-sm text-muted-foreground">
						{textGet("cash.last_close.closed_by", {
							name: session.closed_by_name || "—",
						})}
					</p>
				</div>
				<div className="flex flex-wrap items-center gap-x-4 gap-y-2">
					<div className="flex items-baseline gap-2 text-sm">
						<span className="text-muted-foreground">
							{textGet("cash.summary.counted")}
						</span>
						<span className="font-semibold tabular-nums">
							{formatMoney(session.counted_cash)}
						</span>
					</div>
					<DifferenceBadge value={session.difference} />
					<div className="flex gap-1">
						<SessionReportButton
							sessionId={session.ID}
							variant="ghost"
							size="sm"
						/>
						<Button
							variant="ghost"
							size="sm"
							onClick={() => navigate(`/cash/sessions/${session.ID}`)}
						>
							<Eye className="h-4 w-4" />
							{textGet("cash.last_close.view")}
						</Button>
					</div>
				</div>
			</div>
		</section>
	);
}
