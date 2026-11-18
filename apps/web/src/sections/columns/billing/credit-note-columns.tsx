import type { DataTableColumnDef } from "@pengi/ui";
import { DataTableColumnHeader, Text } from "@pengi/ui";
import {
	type CreditNote,
	downloadCreditNoteRide,
	downloadCreditNoteXml,
	sendCreditNoteEmail,
} from "@/api/billing-service";
import { InvoiceStatusBadge } from "@/components/custom/billing/invoice-status-badge";
import { SriDocumentDownloads } from "@/components/custom/billing/sri-document-downloads";
import {
	SriDocumentEmailAction,
	SriDocumentEmailStatus,
} from "@/components/custom/billing/sri-document-email";
import { Money } from "@/components/custom/formatted";
import { RelativeDate } from "@/components/custom/relative-date";
import { selectColumn } from "@/components/custom/table/select-column";

export function getCreditNoteColumns(
	onRetry: (id: number) => void | Promise<void>,
	onEmailed?: () => void,
): DataTableColumnDef<CreditNote>[] {
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
				return (
					<Money amount={amount} className="font-mono text-right font-medium" />
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
			id: "email",
			meta: { title: "billing.document.email.column" },
			header: () => <Text uuid="billing.document.email.column" />,
			cell: ({ row }) => <SriDocumentEmailStatus doc={row.original} />,
			enableSorting: false,
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
		{
			id: "actions",
			header: () => null,
			cell: ({ row }) => {
				const note = row.original;
				return (
					<div className="flex justify-end gap-1">
						<SriDocumentEmailAction
							doc={note}
							buyerEmail={note.invoice?.patient?.email}
							send={(email) => sendCreditNoteEmail(note.ID, email)}
							onSent={onEmailed}
						/>
						<SriDocumentDownloads
							status={note.status}
							fileBase={`nota_credito_${note.establishment_code}-${note.emission_point_code}-${note.sequential}`}
							accessKey={note.access_key}
							downloadRide={() => downloadCreditNoteRide(note.ID)}
							downloadXml={() => downloadCreditNoteXml(note.ID)}
						/>
					</div>
				);
			},
			enableSorting: false,
			enableHiding: false,
		},
	];
}
