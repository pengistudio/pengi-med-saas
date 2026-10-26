import {
	type Cell,
	type CellContext,
	type Column,
	type ColumnDef,
	columnFilteringFeature,
	columnSizingFeature,
	columnVisibilityFeature,
	createFilteredRowModel,
	createPaginatedRowModel,
	createSortedRowModel,
	filterFns,
	type ReactTable,
	type Row,
	type RowData,
	rowPaginationFeature,
	rowSelectionFeature,
	rowSortingFeature,
	sortFns,
	tableFeatures,
} from "@tanstack/react-table";

/**
 * What every data table supports: column filter (search box), column
 * visibility (Vista menu), sorting, pagination and row selection. One
 * module-level object, so the table's memoization holds across renders.
 */
export const dataTableFeatures = tableFeatures({
	columnFilteringFeature,
	columnSizingFeature,
	columnVisibilityFeature,
	rowSortingFeature,
	rowPaginationFeature,
	rowSelectionFeature,
	filteredRowModel: createFilteredRowModel(),
	sortedRowModel: createSortedRowModel(),
	paginatedRowModel: createPaginatedRowModel(),
	filterFns,
	sortFns,
});

export type DataTableFeatures = typeof dataTableFeatures;

export type DataTableColumnDef<
	TData extends RowData,
	TValue = unknown,
> = ColumnDef<DataTableFeatures, TData, TValue>;
export type DataTableColumn<TData extends RowData, TValue = unknown> = Column<
	DataTableFeatures,
	TData,
	TValue
>;
export type DataTableInstance<TData extends RowData> = ReactTable<
	DataTableFeatures,
	TData
>;
export type DataTableRow<TData extends RowData> = Row<DataTableFeatures, TData>;
export type DataTableCell<TData extends RowData, TValue = unknown> = Cell<
	DataTableFeatures,
	TData,
	TValue
>;
export type DataTableCellContext<
	TData extends RowData,
	TValue = unknown,
> = CellContext<DataTableFeatures, TData, TValue>;
