import { DataTableColumnHeader, Text } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import type { CreditNote } from "@/api/billing-service";
import { InvoiceStatusBadge } from "@/components/custom/billing/invoice-status-badge";
import { RelativeDate } from "@/components/custom/relative-date";
import { selectColumn } from "@/components/custom/table/select-column";

export function getCreditNoteColumns(
	onRetry: (id: number) => void | Promise<void>,
): ColumnDef<CreditNote>[] {
	return [
		selectColumn<CreditNote>(),
		{
			accessorKey: "sequential",
			meta: { title: "billing.credit_note.column.sequential", phone: "title" },
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
			meta: { title: "billing.credit_note.column.invoice" },
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
			meta: { title: "billing.credit_note.column.reason", phone: "subtitle" },
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
				const creditNote = row.original;
				return (
					<InvoiceStatusBadge
						status={creditNote.status}
						errorMessage={creditNote.error_message}
						errorCode={creditNote.error_code}
						onRetry={() => onRetry(creditNote.ID)}
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
