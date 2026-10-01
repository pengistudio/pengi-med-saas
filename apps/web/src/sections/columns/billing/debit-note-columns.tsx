import { DataTableColumnHeader, Text } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import type { DebitNote } from "@/api/billing-service";
import { InvoiceStatusBadge } from "@/components/custom/billing/invoice-status-badge";
import { RelativeDate } from "@/components/custom/relative-date";
import { selectColumn } from "@/components/custom/table/select-column";

export function getDebitNoteColumns(
	onRetry: (id: number) => void | Promise<void>,
): ColumnDef<DebitNote>[] {
	return [
		selectColumn<DebitNote>(),
		{
			accessorKey: "sequential",
			meta: { title: "billing.debit_note.column.sequential", phone: "title" },
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
			meta: { title: "billing.debit_note.column.invoice", phone: "subtitle" },
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
			meta: { title: "billing.invoice.column.total", phone: "end" },
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
			meta: { title: "billing.invoice.column.status", phone: "status" },
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
						errorCode={debitNote.error_code}
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
			meta: { title: "billing.invoice.column.date" },
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
