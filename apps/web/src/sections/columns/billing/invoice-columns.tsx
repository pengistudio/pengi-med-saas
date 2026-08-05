import { Button, Checkbox, DataTableColumnHeader, Text } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import { Download } from "lucide-react";
import { downloadInvoiceRide, type Invoice } from "@/api/billing-service";
import { InvoiceStatusBadge } from "@/components/custom/billing/invoice-status-badge";
import { RelativeDate } from "@/components/custom/relative-date";
import { useText } from "@/hooks/use-text";

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
						{patient ? patient.document : <Text uuid="common.unknown" />}
					</span>
				);
			},
		},
		{
			accessorKey: "patient.full_name",
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
							<Text uuid="common.unknown" />
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
				const invoice = row.original;
				return (
					<InvoiceStatusBadge
						status={invoice.status}
						errorMessage={invoice.error_message}
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

export function getInvoiceColumnsMobile(
	onRetry: (id: number) => void | Promise<void>,
): ColumnDef<Invoice>[] {
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
				const invoice = row.original;
				const amount = new Intl.NumberFormat("en-US", {
					style: "currency",
					currency: "USD",
				}).format(invoice.total);

				return (
					<div className="flex flex-col gap-1 py-1">
						<div className="flex justify-between items-center">
							<span className="font-medium text-sm">
								{invoice.establishment_code}-{invoice.emission_point_code}-
								{invoice.sequential}
							</span>
							<span className="font-mono text-sm font-semibold">{amount}</span>
						</div>
						<div className="flex justify-between items-center text-xs text-muted-foreground">
							<span>
								{invoice.patient ? (
									`${invoice.patient.first_name} ${invoice.patient.last_name}`
								) : (
									<Text uuid="billing.invoice.final_consumer" />
								)}
							</span>
							<InvoiceStatusBadge
								status={invoice.status}
								errorMessage={invoice.error_message}
								onRetry={() => onRetry(invoice.ID)}
							/>
						</div>
						<DownloadRideButton invoice={invoice} />
					</div>
				);
			},
		},
	];
}
