import { useText } from "@pengi/shared";
import type { DataTableColumnDef } from "@pengi/ui";
import { Button, DataTableColumnHeader, Text } from "@pengi/ui";
import { Download, Wallet } from "lucide-react";
import { downloadInvoiceRide, type Invoice } from "@/api/billing-service";
import { InvoiceStatusBadge } from "@/components/custom/billing/invoice-status-badge";
import { Money } from "@/components/custom/formatted";
import { RelativeDate } from "@/components/custom/relative-date";
import { selectColumn } from "@/components/custom/table/select-column";
import { PaymentStatusBadge } from "@/components/features/cash/payment-status-badge";

function DownloadRideButton({ invoice }: { invoice: Invoice }) {
	const { textGet } = useText();

	if (invoice.status !== "authorized") return null;

	async function handleDownload() {
		const response = await downloadInvoiceRide(invoice.ID);
		if (response.success && response.data) {
			const blobUrl = window.URL.createObjectURL(response.data);
			const tempLink = document.createElement("a");
			tempLink.href = blobUrl;
			tempLink.download = `factura_${invoice.establishment_code}-${invoice.emission_point_code}-${invoice.sequential}.pdf`;
			document.body.appendChild(tempLink);
			tempLink.click();
			document.body.removeChild(tempLink);
			window.URL.revokeObjectURL(blobUrl);
		}
	}

	return (
		<Button variant="ghost" size="sm" onClick={handleDownload}>
			<Download className="mr-2 h-4 w-4" />
			{textGet("billing.invoice.ride.download")}
		</Button>
	);
}

function ChargeButton({
	invoice,
	onCharge,
}: {
	invoice: Invoice;
	onCharge: (invoice: Invoice) => void;
}) {
	const { textGet } = useText();
	// Null: issued before the first cash session, never charged at the desk.
	if (
		invoice.payment_status !== "unpaid" &&
		invoice.payment_status !== "partial"
	)
		return null;
	return (
		<Button variant="ghost" size="sm" onClick={() => onCharge(invoice)}>
			<Wallet className="mr-2 h-4 w-4" />
			{textGet("cash.payment.charge")}
		</Button>
	);
}

/**
 * Invoice list columns. `cash` is set when the plan includes the cash desk
 * (adds the collection status); `cash.onCharge` (the Cobrar action) only when
 * the user may charge too.
 */
export function getInvoiceColumns(
	onRetry: (id: number) => void | Promise<void>,
	cash?: { onCharge?: (invoice: Invoice) => void },
): DataTableColumnDef<Invoice>[] {
	const onCharge = cash?.onCharge;
	const paymentStatusColumn: DataTableColumnDef<Invoice> = {
		accessorKey: "payment_status",
		meta: { title: "cash.payment_status.column" },
		header: ({ column }) => (
			<DataTableColumnHeader
				column={column}
				title={<Text uuid="cash.payment_status.column" />}
			/>
		),
		cell: ({ row }) => (
			<PaymentStatusBadge status={row.original.payment_status} />
		),
	};
	return [
		selectColumn<Invoice>(),
		{
			accessorKey: "sequential",
			meta: { title: "billing.invoice.column.sequential", phone: "title" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="billing.invoice.column.sequential" />}
				/>
			),
			cell: ({ row }) => {
				const invoice = row.original;
				return (
					<span className="font-medium">
						{invoice.establishment_code}-{invoice.emission_point_code}-
						{invoice.sequential}
					</span>
				);
			},
		},
		{
			accessorKey: "patient.document",
			meta: { title: "billing.invoice.column.document" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="billing.invoice.column.document" />}
				/>
			),
			cell: ({ row }) => {
				const patient = row.original.patient;
				return (
					<span>
						{patient ? (
							patient.document
						) : (
							<Text uuid="billing.invoice.final_consumer" />
						)}
					</span>
				);
			},
		},
		{
			accessorKey: "patient.full_name",
			meta: { title: "billing.invoice.column.name", phone: "subtitle" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="billing.invoice.column.name" />}
				/>
			),
			cell: ({ row }) => {
				const patient = row.original.patient;
				if (!patient)
					return (
						<span className="text-muted-foreground">
							<Text uuid="billing.invoice.final_consumer" />
						</span>
					);
				return (
					<span>
						{patient.first_name} {patient.last_name}
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
				const invoice = row.original;
				return (
					<InvoiceStatusBadge
						status={invoice.status}
						errorMessage={invoice.error_message}
						errorCode={invoice.error_code}
						onRetry={() => onRetry(invoice.ID)}
					/>
				);
			},
			filterFn: (row, id, value) => {
				return value.includes(row.getValue(id));
			},
		},
		...(cash ? [paymentStatusColumn] : []),
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
			cell: ({ row }) => (
				<div className="flex justify-end gap-1">
					{onCharge && (
						<ChargeButton invoice={row.original} onCharge={onCharge} />
					)}
					<DownloadRideButton invoice={row.original} />
				</div>
			),
			enableSorting: false,
			enableHiding: false,
		},
	];
}
