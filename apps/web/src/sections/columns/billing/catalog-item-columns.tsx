import type { ColumnDef } from "@tanstack/react-table";
import { format } from "date-fns";
import { MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import type { CatalogItem } from "@/api/billing-service";
import { DataTableColumnHeader } from "@/components/custom/table/data-table-column-header";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Text } from "@/components/ui/text";

interface CatalogItemColumnProps {
	onEdit: (id: number) => void;
	onDelete: (id: number) => void;
}

export const getCatalogItemColumns = ({
	onEdit,
	onDelete,
}: CatalogItemColumnProps): ColumnDef<CatalogItem>[] => [
	{
		id: "select",
		header: ({ table }) => (
			<Checkbox
				checked={table.getIsAllPageRowsSelected()}
				onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
				aria-label="Select all"
				className="translate-y-[2px]"
			/>
		),
		cell: ({ row }) => (
			<Checkbox
				checked={row.getIsSelected()}
				onCheckedChange={(value) => row.toggleSelected(!!value)}
				aria-label="Select row"
				className="translate-y-[2px]"
			/>
		),
		enableSorting: false,
		enableHiding: false,
	},
	{
		accessorKey: "sku",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.catalog-item.column.sku" />}
			/>
		),
		cell: ({ row }) => {
			const catalogItem = row.original;
			return <span className="font-medium">{catalogItem.sku}</span>;
		},
	},
	{
		accessorKey: "name",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.catalog-item.column.name" />}
			/>
		),
		cell: ({ row }) => {
			const catalogItem = row.original;
			return <span>{catalogItem.name}</span>;
		},
	},
	{
		accessorKey: "unit_price",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.catalog-item.column.price" />}
			/>
		),
		cell: ({ row }) => {
			const amount = row.original.unit_price;
			const formatted = new Intl.NumberFormat("en-US", {
				style: "currency",
				currency: "USD",
			}).format(amount);
			return (
				<span className="font-mono text-right font-medium">{formatted}</span>
			);
		},
	},
	{
		accessorKey: "createdAt",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.catalog-item.column.date" />}
			/>
		),
		cell: ({ row }) => {
			const raw = row.original.CreatedAt;
			if (!raw) return <span className="text-muted-foreground text-sm">—</span>;
			const date = new Date(raw);
			if (Number.isNaN(date.getTime()))
				return <span className="text-muted-foreground text-sm">—</span>;
			return (
				<span className="text-muted-foreground whitespace-nowrap text-sm">
					{format(date, "dd/MMM/yyyy HH:mm")}
				</span>
			);
		},
	},
	{
		id: "actions",
		cell: ({ row }) => {
			const catalogItem = row.original;
			return (
				<DropdownMenu>
					<DropdownMenuTrigger>
						<Button variant="ghost" className="h-8 w-8 p-0">
							<Text uuid="table.button.open_menu" className="sr-only" />
							<MoreHorizontal className="h-4 w-4" />
						</Button>
					</DropdownMenuTrigger>
					<DropdownMenuContent align="end">
						<DropdownMenuGroup>
							<DropdownMenuLabel>
								<Text uuid="table.actions" />
							</DropdownMenuLabel>
							<DropdownMenuItem onClick={() => onEdit(catalogItem.ID)}>
								<Pencil className="mr-2 h-4 w-4" />
								<Text uuid="billing.catalog-item.list.action.edit" />
							</DropdownMenuItem>
							<DropdownMenuItem
								onClick={() => onDelete(catalogItem.ID)}
								className="text-red-600 focus:text-red-600 focus:bg-red-100 dark:focus:bg-red-900"
							>
								<Trash2 className="mr-2 h-4 w-4" />
								<Text uuid="billing.catalog-item.list.action.delete" />
							</DropdownMenuItem>
						</DropdownMenuGroup>
					</DropdownMenuContent>
				</DropdownMenu>
			);
		},
	},
];

export const getCatalogItemColumnsMobile = ({
	onEdit,
	onDelete,
}: CatalogItemColumnProps): ColumnDef<CatalogItem>[] => [
	{
		id: "select",
		header: ({ table }) => (
			<Checkbox
				checked={table.getIsAllPageRowsSelected()}
				onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
				aria-label="Select all"
				className="translate-y-[2px]"
			/>
		),
		cell: ({ row }) => (
			<Checkbox
				checked={row.getIsSelected()}
				onCheckedChange={(value) => row.toggleSelected(!!value)}
				aria-label="Select row"
				className="translate-y-[2px]"
			/>
		),
		enableSorting: false,
		enableHiding: false,
	},
	{
		accessorKey: "sku",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.catalog-item.column.sku" />}
			/>
		),
		cell: ({ row }) => {
			const catalogItem = row.original;
			return <span className="font-medium">{catalogItem.sku}</span>;
		},
	},
	{
		accessorKey: "name",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.catalog-item.column.name" />}
			/>
		),
		cell: ({ row }) => {
			const catalogItem = row.original;
			return <span>{catalogItem.name}</span>;
		},
	},
	{
		accessorKey: "unit_price",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.catalog-item.column.price" />}
			/>
		),
		cell: ({ row }) => {
			const amount = row.original.unit_price;
			const formatted = new Intl.NumberFormat("en-US", {
				style: "currency",
				currency: "USD",
			}).format(amount);
			return (
				<span className="font-mono text-right font-medium">{formatted}</span>
			);
		},
	},
	{
		accessorKey: "createdAt",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.catalog-item.column.date" />}
			/>
		),
		cell: ({ row }) => {
			const raw = row.original.CreatedAt;
			if (!raw) return <span className="text-muted-foreground text-sm">—</span>;
			const date = new Date(raw);
			if (Number.isNaN(date.getTime()))
				return <span className="text-muted-foreground text-sm">—</span>;
			return (
				<span className="text-muted-foreground whitespace-nowrap text-sm">
					{format(date, "dd/MMM/yyyy HH:mm")}
				</span>
			);
		},
	},
	{
		id: "actions",
		cell: ({ row }) => {
			const catalogItem = row.original;
			return (
				<DropdownMenu>
					<DropdownMenuTrigger>
						<Button variant="ghost" className="h-8 w-8 p-0">
							<Text uuid="table.button.open_menu" className="sr-only" />
							<MoreHorizontal className="h-4 w-4" />
						</Button>
					</DropdownMenuTrigger>
					<DropdownMenuContent align="end">
						<DropdownMenuLabel>
							<Text uuid="table.actions" />
						</DropdownMenuLabel>
						<DropdownMenuItem onClick={() => onEdit(catalogItem.ID)}>
							<Pencil className="mr-2 h-4 w-4" />
							<Text uuid="billing.catalog-item.list.action.edit" />
						</DropdownMenuItem>
						<DropdownMenuItem
							onClick={() => onDelete(catalogItem.ID)}
							className="text-red-600 focus:text-red-600 focus:bg-red-100 dark:focus:bg-red-900"
						>
							<Trash2 className="mr-2 h-4 w-4" />
							<Text uuid="billing.catalog-item.list.action.delete" />
						</DropdownMenuItem>
					</DropdownMenuContent>
				</DropdownMenu>
			);
		},
	},
];
