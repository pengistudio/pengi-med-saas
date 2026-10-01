import { useText } from "@pengi/shared";
import {
	Button,
	DataTablePagination,
	DataTableViewOptions,
	Input,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
	useViewport,
} from "@pengi/ui";
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
import { useRowStore } from "@/store/row-store";
import { DataTablePhoneList } from "./data-table-phone-list";

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
	// Actions on the selected rows. They take the toolbar's place while at
	// least one row is selected, so they never sit idle above the table.
	bulkActions?: React.ReactNode;
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
	bulkActions,
	rowClassName,
	emptyState,
}: DataTableProps<TData, TValue>) {
	const { textGet } = useText();
	const { isPhone } = useViewport();
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
		// Select by record ID, not row position: after a delete or a refetch
		// the selection must not jump to whichever rows took those positions.
		getRowId: (row, index) => {
			const id = (row as { ID?: number }).ID;
			return id === undefined ? String(index) : String(id);
		},
		onRowSelectionChange: setRowSelection,
		onSortingChange: setSorting,
		onColumnFiltersChange: setColumnFilters,
		onColumnVisibilityChange: setColumnVisibility,
		getCoreRowModel: getCoreRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		getSortedRowModel: getSortedRowModel(),
	});

	useEffect(() => {
		setRows(table.getFilteredSelectedRowModel().rows as Row<unknown>[]);
	}, [rowSelection, data, setRows, table]);

	const selectedCount = table.getFilteredSelectedRowModel().rows.length;
	const showBulkActions = bulkActions !== undefined && selectedCount > 0;

	return (
		<div className="space-y-4">
			{showBulkActions ? (
				// Same height as the search row, so the table doesn't jump.
				<div className="flex min-h-[4.25rem] flex-wrap items-center gap-2 py-4">
					<span className="text-sm font-medium">
						{textGet("table.selection.count", { count: selectedCount })}
					</span>
					<Button
						variant="ghost"
						size="sm"
						onClick={() => table.resetRowSelection()}
					>
						{textGet("table.selection.clear")}
					</Button>
					<div className="ml-auto flex flex-wrap items-center gap-2">
						{bulkActions}
					</div>
				</div>
			) : (
				<div className="flex flex-wrap items-center justify-between gap-2 py-4">
					{onSearchChange ? (
						<div className="flex w-full items-center sm:max-w-sm">
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
						<div className="flex w-full items-center sm:max-w-sm">
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
					{/* On a phone the filters get their own line and scroll sideways. */}
					<div className="flex max-w-full items-center gap-2 overflow-x-auto max-sm:w-full sm:ml-auto">
						{toolbarRight}
						<DataTableViewOptions table={table} />
					</div>
				</div>
			)}

			{isPhone ? (
				<DataTablePhoneList
					table={table}
					loading={loading}
					rowClassName={rowClassName}
					emptyState={emptyState}
				/>
			) : (
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
									<TableRow key={i}>
										{columns.map((_, j) => (
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
			)}
			<DataTablePagination table={table} serverSide={isServerPaginated} />
		</div>
	);
}
