import { Badge, Checkbox, DataTableColumnHeader, Text } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import { format } from "date-fns";
import type { CreditNote } from "@/api/billing-service";

const getStatusBadge = (status: string) => {
	switch (status) {
		case "processing":
			return (
				<Badge variant="secondary">
					<Text uuid="billing.status.processing" />
				</Badge>
			);
		case "signed":
		case "validated":
			return (
				<Badge className="bg-blue-500 hover:bg-blue-600 text-white">
					<Text uuid={`billing.status.${status}`} />
				</Badge>
			);
		case "authorized":
			return (
				<Badge className="bg-green-500 hover:bg-green-600 text-white">
					<Text uuid="billing.status.authorized" />
				</Badge>
			);
		default:
			return (
				<Badge variant="outline">
					<Text uuid="billing.status.draft" />
				</Badge>
			);
	}
};

export const creditNoteColumns: ColumnDef<CreditNote>[] = [
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
		accessorKey: "sequential",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.credit_note.column.sequential" />}
			/>
		),
		cell: ({ row }) => {
			const creditNote = row.original;
			return (
				<span className="font-medium">
					{creditNote.establishment_code}-{creditNote.emission_point_code}-
					{creditNote.sequential}
				</span>
			);
		},
	},
	{
		accessorKey: "invoice.sequential",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.credit_note.column.invoice" />}
			/>
		),
		cell: ({ row }) => {
			const invoice = row.original.invoice;
			return (
				<span>
					{invoice
						? `${invoice.establishment_code}-${invoice.emission_point_code}-${invoice.sequential}`
						: "—"}
				</span>
			);
		},
	},
	{
		accessorKey: "reason",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.credit_note.column.reason" />}
			/>
		),
		cell: ({ row }) => (
			<span className="text-muted-foreground line-clamp-1 max-w-xs">
				{row.original.reason}
			</span>
		),
	},
	{
		accessorKey: "total",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.invoice.column.total" />}
			/>
		),
		cell: ({ row }) => {
			const amount = parseFloat(row.getValue("total"));
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
		accessorKey: "status",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.invoice.column.status" />}
			/>
		),
		cell: ({ row }) => {
			const status = row.getValue("status") as string;
			return getStatusBadge(status);
		},
		filterFn: (row, id, value) => {
			return value.includes(row.getValue(id));
		},
	},
	{
		accessorKey: "createdAt",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.invoice.column.date" />}
			/>
		),
		cell: ({ row }) => {
			return (
				<span className="text-muted-foreground whitespace-nowrap text-sm">
					{format(
						new Date(row.original.CreatedAt as string),
						"dd/MMM/yyyy HH:mm",
					)}
				</span>
			);
		},
	},
];

export const creditNoteColumnsMobile: ColumnDef<CreditNote>[] = [
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
		accessorKey: "summary",
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="billing.invoice.column.summary" />}
			/>
		),
		cell: ({ row }) => {
			const creditNote = row.original;
			const amount = new Intl.NumberFormat("en-US", {
				style: "currency",
				currency: "USD",
			}).format(creditNote.total);

			return (
				<div className="flex flex-col gap-1 py-1">
					<div className="flex justify-between items-center">
						<span className="font-medium text-sm">
							{creditNote.establishment_code}-{creditNote.emission_point_code}-
							{creditNote.sequential}
						</span>
						<span className="font-mono text-sm font-semibold">{amount}</span>
					</div>
					<div className="flex justify-between items-center text-xs text-muted-foreground">
						<span className="line-clamp-1">{creditNote.reason}</span>
						{getStatusBadge(creditNote.status)}
					</div>
				</div>
			);
		},
	},
];
