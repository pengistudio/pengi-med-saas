import { useText } from "@pengi/shared";
import { cn } from "@pengi/ui";
import {
	type Cell,
	flexRender,
	type Row,
	type Table,
} from "@tanstack/react-table";
import type * as React from "react";

type Slot =
	| "select"
	| "actions"
	| "title"
	| "end"
	| "subtitle"
	| "status"
	| "detail"
	| "hidden";

type Slots<TData> = Partial<Record<Slot, Cell<TData, unknown>[]>>;

const slotOf = <TData,>(cell: Cell<TData, unknown>): Slot => {
	const { column } = cell;
	if (column.id === "select") return "select";
	if (column.id === "actions") return "actions";
	return column.columnDef.meta?.phone ?? "detail";
};

function slotsOf<TData>(row: Row<TData>): Slots<TData> {
	const slots: Slots<TData> = {};
	for (const cell of row.getVisibleCells()) {
		const slot = slotOf(cell);
		slots[slot] = [...(slots[slot] ?? []), cell];
	}
	// Columns without phone placement still read: the first one leads the row.
	if (!slots.title && slots.detail?.length) {
		const [first, ...rest] = slots.detail;
		slots.title = [first];
		slots.detail = rest;
	}
	// A detail without a value is a "—" taking a line; leave it out.
	slots.detail = slots.detail?.filter((cell) => !isEmpty(cell));
	return slots;
}

const isEmpty = <TData,>(cell: Cell<TData, unknown>) => {
	if (!cell.column.accessorFn) return false;
	const value = cell.getValue();
	return value === null || value === undefined || value === "";
};

const render = <TData,>(cell: Cell<TData, unknown>) =>
	flexRender(cell.column.columnDef.cell, cell.getContext());

const renderAll = <TData,>(cells?: Cell<TData, unknown>[]) =>
	cells?.map((cell) => <span key={cell.id}>{render(cell)}</span>);

interface DataTablePhoneListProps<TData> {
	table: Table<TData>;
	loading?: boolean;
	rowClassName?: (row: Row<TData>) => string;
	emptyState?: React.ReactNode;
}

/**
 * The table reflowed for a phone: one entry per row, columns placed by their
 * `meta.phone` (title and end on the first line, subtitle and status on the
 * second, the rest as labelled details), selection and actions at the edges.
 */
export function DataTablePhoneList<TData>({
	table,
	loading,
	rowClassName,
	emptyState,
}: DataTablePhoneListProps<TData>) {
	const { textGet } = useText();
	const rows = table.getRowModel().rows;
	const selectHeader = table
		.getHeaderGroups()[0]
		?.headers.find((header) => header.column.id === "select");

	return (
		<div className="rounded-md border">
			{selectHeader && rows.length > 0 && !loading && (
				<div className="flex items-center gap-3 border-b px-3 py-2.5 text-sm text-muted-foreground">
					{flexRender(
						selectHeader.column.columnDef.header,
						selectHeader.getContext(),
					)}
					<span>{textGet("table.select_all")}</span>
				</div>
			)}
			<ul className="divide-y">
				{loading ? (
					Array.from({ length: 6 }).map((_, i) => (
						<li key={i} className="space-y-2 px-3 py-3.5">
							<div className="h-4 w-2/3 animate-pulse rounded bg-muted" />
							<div className="h-3 w-1/3 animate-pulse rounded bg-muted" />
						</li>
					))
				) : rows.length ? (
					rows.map((row) => {
						const slots = slotsOf(row);
						return (
							<li
								key={row.id}
								data-state={row.getIsSelected() ? "selected" : undefined}
								className={cn(
									"flex gap-3 px-3 py-3 data-[state=selected]:bg-muted",
									rowClassName?.(row),
								)}
							>
								{slots.select && (
									<div className="flex h-5 shrink-0 items-center">
										{renderAll(slots.select)}
									</div>
								)}
								<div className="min-w-0 flex-1 space-y-1">
									<div className="flex items-start justify-between gap-3">
										<div className="flex min-w-0 flex-wrap items-center gap-x-2 text-sm font-medium">
											{renderAll(slots.title)}
										</div>
										{slots.end && (
											<div className="flex shrink-0 items-center gap-2 text-sm">
												{renderAll(slots.end)}
											</div>
										)}
									</div>
									{(slots.subtitle || slots.status) && (
										<div className="flex items-center justify-between gap-3 text-sm text-muted-foreground">
											<div className="flex min-w-0 flex-wrap items-center gap-x-2">
												{renderAll(slots.subtitle)}
											</div>
											{slots.status && (
												<div className="flex shrink-0 items-center gap-2">
													{renderAll(slots.status)}
												</div>
											)}
										</div>
									)}
									{slots.detail && slots.detail.length > 0 && (
										<div className="flex flex-wrap gap-x-4 gap-y-1 pt-0.5 text-xs">
											{slots.detail.map((cell) => {
												const label = cell.column.columnDef.meta?.title;
												return (
													<span key={cell.id} className="flex min-w-0 gap-1.5">
														{label && (
															<span className="shrink-0 text-muted-foreground">
																{textGet(label)}
															</span>
														)}
														<span className="min-w-0">{render(cell)}</span>
													</span>
												);
											})}
										</div>
									)}
								</div>
								{slots.actions && (
									<div className="-my-1 -mr-1 flex shrink-0 items-start">
										{renderAll(slots.actions)}
									</div>
								)}
							</li>
						);
					})
				) : (
					<li className="flex h-32 items-center justify-center px-3 text-center text-sm">
						{emptyState ?? textGet("table.no_results")}
					</li>
				)}
			</ul>
		</div>
	);
}
