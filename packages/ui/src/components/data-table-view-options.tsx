import type { RowData, Table } from "@tanstack/react-table";
import { SlidersHorizontal } from "lucide-react";

declare module "@tanstack/react-table" {
	interface ColumnMeta<TData extends RowData, TValue> {
		/** i18n key with the column's name, where its header can't render (Vista menu, phone labels). */
		title?: string;
		/**
		 * Where the column goes when the table reflows into a phone list:
		 * `title` and `end` share the first line, `subtitle` and `status` the
		 * second, `detail` (the default) wraps below with its `title` as label.
		 * The `select` and `actions` columns are placed by id.
		 */
		phone?: "title" | "end" | "subtitle" | "status" | "detail" | "hidden";
	}
}

import { Button } from "./button";
import {
	DropdownMenu,
	DropdownMenuCheckboxItem,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "./dropdown-menu";
import { Text } from "./text";

interface DataTableViewOptionsProps<TData> {
	table: Table<TData>;
}

export function DataTableViewOptions<TData>({
	table,
}: DataTableViewOptionsProps<TData>) {
	return (
		<DropdownMenu>
			<DropdownMenuTrigger
				render={
					<Button
						variant="outline"
						size="sm"
						className="ml-auto hidden h-8 lg:flex"
					/>
				}
			>
				<SlidersHorizontal className="mr-2 h-4 w-4" />
				<Text uuid="table.view_options.view" />
			</DropdownMenuTrigger>
			<DropdownMenuContent align="end" className="w-[150px]">
				<DropdownMenuGroup>
					<DropdownMenuLabel>
						<Text uuid="table.view_options.toggle_columns" />
					</DropdownMenuLabel>
					<DropdownMenuSeparator />
					{table
						.getAllColumns()
						.filter(
							(column) =>
								typeof column.accessorFn !== "undefined" && column.getCanHide(),
						)
						.map((column) => {
							const title = column.columnDef.meta?.title;

							return (
								<DropdownMenuCheckboxItem
									key={column.id}
									className="capitalize"
									checked={column.getIsVisible()}
									onCheckedChange={(value) => column.toggleVisibility(!!value)}
								>
									{title ? <Text uuid={title as string} /> : column.id}
								</DropdownMenuCheckboxItem>
							);
						})}
				</DropdownMenuGroup>
			</DropdownMenuContent>
		</DropdownMenu>
	);
}
