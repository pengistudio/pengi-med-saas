import { useText } from "@pengi/shared";
import { Button, DataTableColumnHeader, Text } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import { Download } from "lucide-react";
import { downloadInvoiceRide, type Invoice } from "@/api/billing-service";
import { InvoiceStatusBadge } from "@/components/custom/billing/invoice-status-badge";
import { Money } from "@/components/custom/formatted";
import { RelativeDate } from "@/components/custom/relative-date";
import { selectColumn } from "@/components/custom/table/select-column";

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

export function getInvoiceColumns(
	onRetry: (id: number) => void | Promise<void>,
): ColumnDef<Invoice>[] {
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
			cell: ({ row }) => <DownloadRideButton invoice={row.original} />,
			enableSorting: false,
			enableHiding: false,
		},
	];
}
