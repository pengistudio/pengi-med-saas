import {
	type ColumnDef,
	type ColumnFiltersState,
	flexRender,
	getCoreRowModel,
	getFilteredRowModel,
	getPaginationRowModel,
	getSortedRowModel,
	type Row,
	type SortingState,
	useReactTable,
	type VisibilityState,
} from "@tanstack/react-table";
import * as React from "react";
import { useEffect } from "react";
import { Input } from "@/components/ui/input";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { useText } from "@/hooks/use-text";
import { useRowStore } from "@/store/row-store";
import { DataTablePagination } from "./data-table-pagination";
import { DataTableViewOptions } from "./data-table-view-options";

interface DataTableProps<TData, TValue> {
	columns: ColumnDef<TData, TValue>[];
	data: TData[];
	loading?: boolean;
	searchKey?: string;
	searchPlaceholder?: string;
	// Server-side search
	searchValue?: string;
	onSearchChange?: (value: string) => void;
	// Server-side pagination
	pageCount?: number;
	page?: number;
	onPageChange?: (page: number) => void;
	// Extra content rendered between search and Vista button
	toolbarRight?: React.ReactNode;
	// Optional per-row className
	rowClassName?: (row: Row<TData>) => string;
	// Optional custom empty state (replaces default "no results" text)
	emptyState?: React.ReactNode;
}

export function DataTable<TData, TValue>({
	columns,
	data,
	loading,
	searchKey,
	searchPlaceholder,
	searchValue,
	onSearchChange,
	pageCount,
	page,
	onPageChange,
	toolbarRight,
	rowClassName,
	emptyState,
}: DataTableProps<TData, TValue>) {
	const { textGet } = useText();
	const [sorting, setSorting] = React.useState<SortingState>([]);
	const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>(
		[],
	);
	const [columnVisibility, setColumnVisibility] =
		React.useState<VisibilityState>({});
	const [rowSelection, setRowSelection] = React.useState({});
	const { setRows } = useRowStore();

	const isServerPaginated =
		pageCount !== undefined && onPageChange !== undefined;

	const table = useReactTable({
		data,
		columns,
		state: {
			sorting,
			columnFilters,
			columnVisibility,
			rowSelection,
			...(isServerPaginated && {
				pagination: { pageIndex: (page ?? 1) - 1, pageSize: 20 },
			}),
		},
		...(isServerPaginated && {
			manualPagination: true,
			pageCount,
			onPaginationChange: (updater) => {
				const prev = { pageIndex: (page ?? 1) - 1, pageSize: 20 };
				const next = typeof updater === "function" ? updater(prev) : updater;
				onPageChange(next.pageIndex + 1);
			},
		}),
		enableRowSelection: true,
		onRowSelectionChange: setRowSelection,
		onSortingChange: setSorting,
		onColumnFiltersChange: setColumnFilters,
		onColumnVisibilityChange: setColumnVisibility,
		getCoreRowModel: getCoreRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		getSortedRowModel: getSortedRowModel(),
	});

	// biome-ignore lint/correctness/useExhaustiveDependencies: triggers on selection change
	useEffect(() => {
		setRows(table.getFilteredSelectedRowModel().rows as Row<unknown>[]);
	}, [rowSelection, setRows, table]);

	return (
		<div className="space-y-4">
			<div className="flex items-center gap-2 justify-between">
				{onSearchChange ? (
					<div className="flex items-center py-4 max-w-sm w-full">
						<Input
							placeholder={
								searchPlaceholder || textGet("table.search.placeholder")
							}
							value={searchValue ?? ""}
							onChange={(event) => onSearchChange(event.target.value)}
							className="max-w-sm w-full"
						/>
					</div>
				) : searchKey ? (
					<div className="flex items-center py-4 max-w-sm w-full">
						<Input
							placeholder={
								searchPlaceholder || textGet("table.search.placeholder")
							}
							value={
								(table.getColumn(searchKey)?.getFilterValue() as string) ?? ""
							}
							onChange={(event) =>
								table.getColumn(searchKey)?.setFilterValue(event.target.value)
							}
							className="max-w-sm w-full"
						/>
					</div>
				) : null}
				<div className="flex items-center gap-2 ml-auto">
					{toolbarRight}
					<DataTableViewOptions table={table} />
				</div>
			</div>

			<div className="rounded-md border">
				<Table>
					<TableHeader>
						{table.getHeaderGroups().map((headerGroup) => (
							<TableRow key={headerGroup.id}>
								{headerGroup.headers.map((header) => {
									return (
										<TableHead key={header.id}>
											{header.isPlaceholder
												? null
												: flexRender(
														header.column.columnDef.header,
														header.getContext(),
													)}
										</TableHead>
									);
								})}
							</TableRow>
						))}
					</TableHeader>
					<TableBody>
						{loading ? (
							Array.from({ length: 6 }).map((_, i) => (
								// biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows
								<TableRow key={i}>
									{columns.map((_, j) => (
										// biome-ignore lint/suspicious/noArrayIndexKey: skeleton cells
										<TableCell key={j}>
											<div className="h-4 w-full animate-pulse rounded bg-muted" />
										</TableCell>
									))}
								</TableRow>
							))
						) : table.getRowModel().rows?.length ? (
							table.getRowModel().rows.map((row) => (
								<TableRow
									key={row.id}
									data-state={row.getIsSelected() && "selected"}
									className={rowClassName?.(row)}
								>
									{row.getVisibleCells().map((cell) => (
										<TableCell key={cell.id}>
											{flexRender(
												cell.column.columnDef.cell,
												cell.getContext(),
											)}
										</TableCell>
									))}
								</TableRow>
							))
						) : (
							<TableRow>
								<TableCell
									colSpan={columns.length}
									className="h-32 text-center"
								>
									{emptyState ?? textGet("table.no_results")}
								</TableCell>
							</TableRow>
						)}
					</TableBody>
				</Table>
			</div>
			<DataTablePagination table={table} serverSide={isServerPaginated} />
		</div>
	);
}
