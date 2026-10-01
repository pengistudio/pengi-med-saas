import { useText } from "@pengi/shared";
import { Checkbox } from "@pengi/ui";
import type { ColumnDef, Row, Table } from "@tanstack/react-table";

function SelectAll<TData>({ table }: { table: Table<TData> }) {
	const { textGet } = useText();
	return (
		<Checkbox
			aria-label={textGet("table.select_all")}
			checked={table.getIsAllPageRowsSelected()}
			indeterminate={table.getIsSomePageRowsSelected()}
			onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
		/>
	);
}

function SelectRow<TData>({ row }: { row: Row<TData> }) {
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
export function selectColumn<TData>(): ColumnDef<TData> {
	return {
		id: "select",
		header: ({ table }) => <SelectAll table={table} />,
		cell: ({ row }) => <SelectRow row={row} />,
		enableSorting: false,
		enableHiding: false,
		size: 50,
	};
}
