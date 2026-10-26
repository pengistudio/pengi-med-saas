import { useText } from "@pengi/shared";
import { cn } from "@pengi/ui";
import type React from "react";

/**
 * The date column of every list: the day large, the month under it, like a
 * page of an appointment booklet. `date` is a YYYY-MM-DD string.
 */
export function DateTile({
	date,
	className,
}: {
	date: string;
	className?: string;
}) {
	const { formatDate } = useText();
	return (
		<div
			className={cn(
				"flex w-11 shrink-0 flex-col items-center leading-none tabular-nums",
				className,
			)}
		>
			<span className="text-2xl font-semibold tracking-tight">
				{Number(date.slice(8, 10))}
			</span>
			<span className="mt-1 text-xs text-muted-foreground">
				{formatDate(date, "month")}
			</span>
		</div>
	);
}

/** A list of rows separated by hairlines; no card around each row. */
export function Ledger({ children }: { children: React.ReactNode }) {
	return <ul className="divide-y divide-border">{children}</ul>;
}

/** One row: the date column, the content, and an optional action at the end. */
export function LedgerRow({
	date,
	action,
	muted = false,
	children,
}: {
	date: string;
	action?: React.ReactNode;
	/** Past or cancelled: the row steps back. */
	muted?: boolean;
	children: React.ReactNode;
}) {
	return (
		<li className="flex items-start gap-4 py-4">
			<DateTile date={date} className={cn(muted && "opacity-60")} />
			<div className="min-w-0 flex-1 space-y-0.5">{children}</div>
			{action && <div className="shrink-0 self-center">{action}</div>}
		</li>
	);
}

/** Splits a list already sorted by date into [year, items] runs. */
export function byYear<T extends { date: string }>(
	items: T[],
): [string, T[]][] {
	const groups: [string, T[]][] = [];
	for (const item of items) {
		const year = item.date.slice(0, 4);
		const last = groups.at(-1);
		if (last && last[0] === year) last[1].push(item);
		else groups.push([year, [item]]);
	}
	return groups;
}

/** A year heading between rows, only when the list spans more than one. */
export function LedgerYear({ year }: { year: string }) {
	return (
		<h3 className="pt-6 pb-1 text-sm font-medium text-muted-foreground tabular-nums">
			{year}
		</h3>
	);
}
