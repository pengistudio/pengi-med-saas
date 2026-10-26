import { useText } from "@pengi/shared";
import type {
	DataTableColumnDef,
	DataTableFeatures,
	DataTableRow,
} from "@pengi/ui";
import { Checkbox } from "@pengi/ui";
import type { RowData, Table } from "@tanstack/react-table";

function SelectAll<TData extends RowData>({
	table,
}: {
	table: Table<DataTableFeatures, TData>;
}) {
	const { textGet } = useText();
	return (
		<Checkbox
			aria-label={textGet("table.select_all")}
			checked={table.getIsAllPageRowsSelected()}
			// v9: "some" is also true when all are selected.
			indeterminate={
				table.getIsSomePageRowsSelected() && !table.getIsAllPageRowsSelected()
			}
			onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
		/>
	);
}

function SelectRow<TData extends RowData>({
	row,
}: {
	row: DataTableRow<TData>;
}) {
	const { textGet } = useText();
	return (
		<Checkbox
			aria-label={textGet("table.select_row")}
			checked={row.getIsSelected()}
			onCheckedChange={(value) => row.toggleSelected(!!value)}
			// Rows may react to clicks; ticking the box shouldn't open the row.
			onClick={(event) => event.stopPropagation()}
		/>
	);
}

/**
 * The checkbox column for multi-select. DataTable recognises it by its
 * `select` id: the bulk actions and the phone list's "select all" rely on it.
 */
export function selectColumn<
	TData extends RowData,
>(): DataTableColumnDef<TData> {
	return {
		id: "select",
		header: ({ table }) => <SelectAll table={table} />,
		cell: ({ row }) => <SelectRow row={row} />,
		enableSorting: false,
		enableHiding: false,
		size: 50,
	};
}
