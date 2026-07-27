import { Checkbox, DataTableColumnHeader, Text } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import type { DebitNote } from "@/api/billing-service";
import { InvoiceStatusBadge } from "@/components/custom/billing/invoice-status-badge";
import { RelativeDate } from "@/components/custom/relative-date";

export function getDebitNoteColumns(
	onRetry: (id: number) => void | Promise<void>,
): ColumnDef<DebitNote>[] {
	return [
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
					title={<Text uuid="billing.debit_note.column.sequential" />}
				/>
			),
			cell: ({ row }) => {
				const debitNote = row.original;
				return (
					<span className="font-medium">
						{debitNote.establishment_code}-{debitNote.emission_point_code}-
						{debitNote.sequential}
					</span>
				);
			},
		},
		{
			accessorKey: "invoice.sequential",
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="billing.debit_note.column.invoice" />}
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
				const debitNote = row.original;
				return (
					<InvoiceStatusBadge
						status={debitNote.status}
						errorMessage={debitNote.error_message}
						onRetry={() => onRetry(debitNote.ID)}
					/>
				);
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
			cell: ({ row }) => (
				<RelativeDate date={row.original.CreatedAt as string} />
			),
		},
	];
}

export function getDebitNoteColumnsMobile(
	onRetry: (id: number) => void | Promise<void>,
): ColumnDef<DebitNote>[] {
	return [
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
				const debitNote = row.original;
				const amount = new Intl.NumberFormat("en-US", {
					style: "currency",
					currency: "USD",
				}).format(debitNote.total);

				return (
					<div className="flex flex-col gap-1 py-1">
						<div className="flex justify-between items-center">
							<span className="font-medium text-sm">
								{debitNote.establishment_code}-{debitNote.emission_point_code}-
								{debitNote.sequential}
							</span>
							<span className="font-mono text-sm font-semibold">{amount}</span>
						</div>
						<div className="flex justify-between items-center text-xs text-muted-foreground">
							<span>
								{debitNote.motives?.[0]?.reason ?? ""}
								{debitNote.motives && debitNote.motives.length > 1
									? ` +${debitNote.motives.length - 1}`
									: ""}
							</span>
							<InvoiceStatusBadge
								status={debitNote.status}
								errorMessage={debitNote.error_message}
								onRetry={() => onRetry(debitNote.ID)}
							/>
						</div>
					</div>
				);
			},
		},
	];
}
