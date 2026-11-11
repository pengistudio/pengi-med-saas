import { useText } from "@pengi/shared";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";
import { Ban } from "lucide-react";
import { useState } from "react";
import { type PaymentView, voidPayment } from "@/api/cash-service";
import {
	VoidPaymentForm,
	type VoidPaymentValues,
} from "@/sections/forms/cash/void-payment-form";

/** Voids a payment of the open session; the dialog is open while set. */
export function VoidPaymentDialog({
	payment,
	onOpenChange,
	onVoided,
}: {
	payment: PaymentView | null;
	onOpenChange: (open: boolean) => void;
	onVoided: () => void;
}) {
	const { textGet, formatMoney } = useText();
	const [loading, setLoading] = useState(false);

	async function handleSubmit(values: VoidPaymentValues) {
		if (!payment) return;
		setLoading(true);
		const res = await voidPayment(payment.ID, values.reason);
		setLoading(false);
		if (res.success) {
			onOpenChange(false);
			onVoided();
		}
	}

	return (
		<Dialog open={Boolean(payment)} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[440px]">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-destructive/10 text-destructive">
							<Ban aria-hidden className="h-4 w-4" />
						</span>
						{textGet("cash.void.title")}
					</DialogTitle>
					<DialogDescription>
						{payment &&
							textGet("cash.void.description", {
								amount: formatMoney(payment.amount),
								invoice: payment.invoice_number,
							})}
					</DialogDescription>
				</DialogHeader>
				{payment && (
					<VoidPaymentForm
						key={payment.ID}
						loading={loading}
						onSubmit={handleSubmit}
						onCancel={() => onOpenChange(false)}
					/>
				)}
			</DialogContent>
		</Dialog>
	);
}
