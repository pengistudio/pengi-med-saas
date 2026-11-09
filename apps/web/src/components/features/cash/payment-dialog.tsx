import { useText } from "@pengi/shared";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
	Skeleton,
} from "@pengi/ui";
import {
	CircleAlert,
	CircleCheck,
	Lock,
	ReceiptText,
	Wallet,
} from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import {
	createPayment,
	getInvoicePayments,
	type InvoicePayments,
} from "@/api/cash-service";
import {
	PaymentForm,
	type PaymentValues,
} from "@/sections/forms/cash/payment-form";
import { useCashStore } from "@/store/cash-store";
import { CashEmptyState } from "./cash-figures";

/** The invoice to charge: its ID and the number shown to the user. */
export interface ChargeTarget {
	id: number;
	number: string;
}

/**
 * Charges an invoice in the open cash session (mixed methods, never more than
 * the balance). With the desk closed it says so and links to /cash.
 */
export function PaymentDialog({
	invoice,
	onOpenChange,
	onPaid,
	cancelLabel,
}: {
	/** The dialog is open while set. */
	invoice: ChargeTarget | null;
	onOpenChange: (open: boolean) => void;
	onPaid?: (result: InvoicePayments) => void;
	cancelLabel?: string;
}) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const session = useCashStore((s) => s.session);
	const fetchCurrent = useCashStore((s) => s.fetchCurrent);
	const [payments, setPayments] = useState<InvoicePayments | null>(null);
	const [ready, setReady] = useState(false);
	const [saving, setSaving] = useState(false);
	const invoiceId = invoice?.id;

	// Each opening re-reads the desk (it may have been closed meanwhile) and
	// the invoice's balance.
	useEffect(() => {
		if (!invoiceId) return;
		let cancelled = false;
		setReady(false);
		setPayments(null);
		Promise.all([fetchCurrent(), getInvoicePayments(invoiceId)]).then(
			([, res]) => {
				if (cancelled) return;
				setPayments(res.success ? res.data : null);
				setReady(true);
			},
		);
		return () => {
			cancelled = true;
		};
	}, [invoiceId, fetchCurrent]);

	async function handleSubmit(values: PaymentValues) {
		if (!invoiceId) return;
		setSaving(true);
		const res = await createPayment(
			invoiceId,
			values.lines.map((l) => ({
				method: l.method,
				amount: l.amount,
				// The reference field is hidden for cash; a value typed before
				// switching the line to cash is not sent.
				reference:
					l.method === "cash" ? undefined : l.reference?.trim() || undefined,
			})),
		);
		setSaving(false);
		if (res.success) {
			onOpenChange(false);
			onPaid?.(res.data);
		}
	}

	const balanceCents = payments ? Math.round(payments.balance * 100) : 0;

	return (
		<Dialog open={Boolean(invoice)} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[640px]">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
							<ReceiptText aria-hidden className="h-4 w-4" />
						</span>
						{textGet("cash.payment.title")}
					</DialogTitle>
					<DialogDescription>
						{invoice &&
							textGet("cash.payment.description", { invoice: invoice.number })}
					</DialogDescription>
				</DialogHeader>
				{!ready ? (
					<div className="space-y-4" aria-hidden>
						<Skeleton className="h-28 w-full rounded-lg" />
						<Skeleton className="h-20 w-full rounded-lg" />
						<div className="flex justify-end gap-2">
							<Skeleton className="h-8 w-24" />
							<Skeleton className="h-9 w-28" />
						</div>
					</div>
				) : !session ? (
					<div className="space-y-4">
						<p
							role="alert"
							className="flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-800 dark:text-amber-300"
						>
							<Lock aria-hidden className="mt-0.5 h-4 w-4 shrink-0" />
							{textGet("cash.payment.desk_closed")}
						</p>
						<div className="flex justify-end gap-2">
							<Button variant="outline" onClick={() => onOpenChange(false)}>
								{cancelLabel ?? textGet("common.cancel")}
							</Button>
							<Button
								onClick={() => {
									onOpenChange(false);
									navigate("/cash");
								}}
							>
								<Wallet className="h-4 w-4" />
								{textGet("cash.payment.go_to_cash")}
							</Button>
						</div>
					</div>
				) : !payments ? (
					<p
						role="alert"
						className="flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm"
					>
						<CircleAlert
							aria-hidden
							className="mt-0.5 h-4 w-4 shrink-0 text-destructive"
						/>
						{textGet("cash.error.load")}
					</p>
				) : balanceCents <= 0 ? (
					<CashEmptyState
						icon={CircleCheck}
						title={textGet("cash.payment.fully_paid")}
					/>
				) : (
					<PaymentForm
						key={invoiceId}
						balance={payments.balance}
						loading={saving}
						cancelLabel={cancelLabel}
						onSubmit={handleSubmit}
						onCancel={() => onOpenChange(false)}
					/>
				)}
			</DialogContent>
		</Dialog>
	);
}
