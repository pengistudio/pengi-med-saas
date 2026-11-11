import { useText } from "@pengi/shared";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";
import { Undo2 } from "lucide-react";
import { useState } from "react";
import { type PaymentView, refundPayment } from "@/api/cash-service";
import {
	RefundPaymentForm,
	type RefundPaymentValues,
} from "@/sections/forms/cash/refund-payment-form";

/**
 * Refunds a payment of a closed session; the refund is recorded in the open
 * session. The dialog is open while `payment` is set.
 */
export function RefundPaymentDialog({
	payment,
	onOpenChange,
	onRefunded,
}: {
	payment: PaymentView | null;
	onOpenChange: (open: boolean) => void;
	onRefunded: () => void;
}) {
	const { textGet, formatMoney } = useText();
	const [loading, setLoading] = useState(false);

	async function handleSubmit(values: RefundPaymentValues) {
		if (!payment) return;
		setLoading(true);
		const res = await refundPayment(payment.ID, values);
		setLoading(false);
		if (res.success) {
			onOpenChange(false);
			onRefunded();
		}
	}

	return (
		<Dialog open={Boolean(payment)} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[440px]">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
							<Undo2 aria-hidden className="h-4 w-4" />
						</span>
						{textGet("cash.refund.title")}
					</DialogTitle>
					<DialogDescription>
						{payment &&
							textGet("cash.refund.description", {
								amount: formatMoney(payment.amount),
								invoice: payment.invoice_number,
							})}
					</DialogDescription>
				</DialogHeader>
				{payment && (
					<RefundPaymentForm
						key={payment.ID}
						max={payment.amount}
						method={payment.method}
						loading={loading}
						onSubmit={handleSubmit}
						onCancel={() => onOpenChange(false)}
					/>
				)}
			</DialogContent>
		</Dialog>
	);
}
