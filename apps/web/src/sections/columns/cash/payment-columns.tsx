import { useText } from "@pengi/shared";
import type { DataTableColumnDef } from "@pengi/ui";
import { Badge, Button, DataTableColumnHeader, Text } from "@pengi/ui";
import { Ban, Undo2 } from "lucide-react";
import type { PaymentView } from "@/api/cash-service";
import { FormattedDate, Money } from "@/components/custom/formatted";
import { MethodIcon } from "@/components/features/cash/cash-figures";
import { paymentMethodLabel } from "@/components/features/cash/cash-labels";
import { cn } from "@/lib/utils";

function PaymentState({ payment }: { payment: PaymentView }) {
	const { textGet } = useText();
	if (payment.status === "voided") {
		return (
			<Badge variant="destructive" title={payment.void_reason || undefined}>
				{textGet("cash.payment.state.voided")}
			</Badge>
		);
	}
	if (payment.kind === "refund") {
		return (
			<Badge variant="secondary" title={payment.reason || undefined}>
				{textGet("cash.payment.state.refund")}
			</Badge>
		);
	}
	return <Badge variant="outline">{textGet("cash.payment.state.valid")}</Badge>;
}

function MethodLabel({ method }: { method: PaymentView["method"] }) {
	const { textGet } = useText();
	return (
		<span className="inline-flex items-center gap-1.5">
			<MethodIcon method={method} className="text-muted-foreground" />
			{paymentMethodLabel(textGet, method)}
		</span>
	);
}

function PaymentActions({
	payment,
	onVoid,
	onRefund,
}: {
	payment: PaymentView;
	onVoid?: (payment: PaymentView) => void;
	onRefund?: (payment: PaymentView) => void;
}) {
	const { textGet } = useText();
	if (payment.status !== "valid") return null;
	return (
		<div className="flex justify-end gap-1">
			{onVoid && (
				<Button variant="ghost" size="sm" onClick={() => onVoid(payment)}>
					<Ban className="h-4 w-4" />
					{textGet("cash.void.action")}
				</Button>
			)}
			{onRefund && payment.kind === "payment" && (
				<Button variant="ghost" size="sm" onClick={() => onRefund(payment)}>
					<Undo2 className="h-4 w-4" />
					{textGet("cash.refund.action")}
				</Button>
			)}
		</div>
	);
}

/**
 * Columns of a session's payments. `onVoid` is passed only for the open
 * session and `onRefund` only for a closed one, each when the user may.
 */
export function getPaymentColumns({
	onVoid,
	onRefund,
}: {
	onVoid?: (payment: PaymentView) => void;
	onRefund?: (payment: PaymentView) => void;
} = {}): DataTableColumnDef<PaymentView>[] {
	const columns: DataTableColumnDef<PaymentView>[] = [
		{
			accessorKey: "CreatedAt",
			meta: { title: "cash.payment.column.time" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.payment.column.time" />}
				/>
			),
			cell: ({ row }) => (
				<FormattedDate value={row.original.CreatedAt} withTime />
			),
		},
		{
			accessorKey: "invoice_number",
			meta: { title: "cash.payment.column.invoice", phone: "title" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.payment.column.invoice" />}
				/>
			),
			cell: ({ row }) => (
				<span className="font-medium">
					{row.original.invoice_number || "—"}
				</span>
			),
		},
		{
			accessorKey: "method",
			meta: { title: "cash.payment.column.method", phone: "subtitle" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.payment.column.method" />}
				/>
			),
			cell: ({ row }) => <MethodLabel method={row.original.method} />,
		},
		{
			accessorKey: "reference",
			meta: { title: "cash.payment.column.reference" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.payment.column.reference" />}
				/>
			),
			cell: ({ row }) => (
				<span className="text-muted-foreground">
					{row.original.reference || "—"}
				</span>
			),
		},
		{
			accessorKey: "amount",
			meta: { title: "cash.payment.column.amount", phone: "end" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.payment.column.amount" />}
				/>
			),
			cell: ({ row }) => (
				<Money
					amount={row.original.amount}
					className={cn(
						"font-medium tabular-nums",
						row.original.status === "voided" &&
							"text-muted-foreground line-through",
					)}
				/>
			),
		},
		{
			accessorKey: "status",
			meta: { title: "cash.payment.column.status", phone: "status" },
			header: ({ column }) => (
				<DataTableColumnHeader
					column={column}
					title={<Text uuid="cash.payment.column.status" />}
				/>
			),
			cell: ({ row }) => <PaymentState payment={row.original} />,
		},
	];
	if (onVoid || onRefund) {
		columns.push({
			id: "actions",
			header: () => null,
			cell: ({ row }) => (
				<PaymentActions
					payment={row.original}
					onVoid={onVoid}
					onRefund={onRefund}
				/>
			),
			enableSorting: false,
			enableHiding: false,
		});
	}
	return columns;
}
