import type { DataTableColumnDef } from "@pengi/ui";
import {
	Button,
	DataTableColumnHeader,
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuTrigger,
	Text,
} from "@pengi/ui";
import { MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import type { Customer } from "@/api/billing-service";
import { FormattedDate } from "@/components/custom/formatted";
import { selectColumn } from "@/components/custom/table/select-column";

interface CustomerColumnProps {
	onEdit: (id: number) => void;
	onDelete: (id: number) => void;
}

export const getCustomerColumns = ({
	onEdit,
	onDelete,
}: CustomerColumnProps): DataTableColumnDef<Customer>[] => [
	selectColumn<Customer>(),
	{
		accessorKey: "identification",
		meta: {
			title: "billing.customer.column.identification",
			phone: "subtitle",
		},
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.customer.column.identification" />}
			/>
		),
		cell: ({ row }) => (
			<span className="font-medium">{row.original.identification}</span>
		),
	},
	{
		accessorKey: "name",
		meta: { title: "billing.customer.column.name", phone: "title" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.customer.column.name" />}
			/>
		),
		cell: ({ row }) => <span>{row.original.name}</span>,
	},
	{
		accessorKey: "email",
		meta: { title: "billing.customer.column.email" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.customer.column.email" />}
			/>
		),
		cell: ({ row }) => <span>{row.original.email || "-"}</span>,
	},
	{
		accessorKey: "phone",
		meta: { title: "billing.customer.column.phone" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.customer.column.phone" />}
			/>
		),
		cell: ({ row }) => <span>{row.original.phone || "-"}</span>,
	},
	{
		accessorKey: "createdAt",
		meta: { title: "billing.customer.column.date" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.customer.column.date" />}
			/>
		),
		cell: ({ row }) => (
			<FormattedDate
				value={row.original.CreatedAt}
				withTime
				className="text-muted-foreground whitespace-nowrap text-sm"
			/>
		),
	},
	{
		id: "actions",
		cell: ({ row }) => {
			const customer = row.original;
			return (
				<DropdownMenu>
					<DropdownMenuTrigger
						render={
							<Button variant="ghost" className="h-8 w-8 p-0">
								<Text uuid="table.button.open_menu" className="sr-only" />
								<MoreHorizontal className="h-4 w-4" />
							</Button>
						}
					/>
					<DropdownMenuContent align="end">
						<DropdownMenuGroup>
							<DropdownMenuLabel>
								<Text uuid="table.actions" />
							</DropdownMenuLabel>
							<DropdownMenuItem onClick={() => onEdit(customer.ID)}>
								<Pencil className="mr-2 h-4 w-4" />
								<Text uuid="billing.customer.list.action.edit" />
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={() => onDelete(customer.ID)}
								className="text-red-600 focus:text-red-600 focus:bg-red-100 dark:focus:bg-red-900"
							>
								<Trash2 className="mr-2 h-4 w-4" />
								<Text uuid="billing.customer.list.action.delete" />
							</DropdownMenuItem>
						</DropdownMenuGroup>
					</DropdownMenuContent>
				</DropdownMenu>
			);
		},
	},
];
