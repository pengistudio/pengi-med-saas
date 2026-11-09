import { useText } from "@pengi/shared";
import { Button, Form, FormInput, FormSelect, FormTextArea } from "@pengi/ui";
import { DollarSign, Loader2, Undo2 } from "lucide-react";
import { useMemo } from "react";
import { z } from "zod";
import { PAYMENT_METHODS, type PaymentMethod } from "@/api/cash-service";
import { MethodIcon } from "@/components/features/cash/cash-figures";
import { paymentMethodLabel } from "@/components/features/cash/cash-labels";

const toCents = (v: number) => Math.round(v * 100);

function refundSchema(max: number) {
	return z.object({
		amount: z
			.number({ error: "cash.form.error.amount" })
			.positive("cash.form.error.amount")
			.refine((v) => toCents(v) <= toCents(max), "cash.refund.error.exceeds"),
		method: z.enum(PAYMENT_METHODS, { error: "cash.form.error.required" }),
		reason: z
			.string()
			.trim()
			.min(1, "cash.form.error.reason_required")
			.max(1000, "cash.form.error.too_long"),
	});
}

export type RefundPaymentValues = z.infer<ReturnType<typeof refundSchema>>;

/**
 * Gives back part or all of a payment from a closed session, as a refund in
 * the open one. `max` is the original amount; the API also subtracts what was
 * already refunded.
 */
export function RefundPaymentForm({
	max,
	method,
	loading,
	onSubmit,
	onCancel,
}: {
	max: number;
	method: PaymentMethod;
	loading?: boolean;
	onSubmit: (values: RefundPaymentValues) => void;
	onCancel: () => void;
}) {
	const { textGet } = useText();
	const schema = useMemo(() => refundSchema(max), [max]);

	return (
		<Form
			schema={schema}
			defaultValues={{ amount: max, method, reason: "" }}
			onSubmit={onSubmit}
		>
			{(field) => (
				<div className="space-y-4">
					<FormInput
						field={field}
						name="amount"
						type="number"
						step="0.01"
						min={0}
						inputMode="decimal"
						label={textGet("cash.refund.amount")}
						startAddon={<DollarSign aria-hidden className="h-4 w-4" />}
						className="tabular-nums"
						autoFocus
					/>
					<FormSelect
						field={field}
						name="method"
						label={textGet("cash.payment.method")}
						options={PAYMENT_METHODS.map((m) => ({
							value: m,
							label: (
								<span className="flex items-center gap-2">
									<MethodIcon method={m} className="text-muted-foreground" />
									{paymentMethodLabel(textGet, m)}
								</span>
							),
						}))}
					/>
					<FormTextArea
						field={field}
						name="reason"
						label={textGet("cash.refund.reason")}
						placeholder={textGet("cash.refund.reason.placeholder")}
						maxLength={1000}
					/>
					<div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
						<Button type="button" variant="outline" onClick={onCancel}>
							{textGet("common.cancel")}
						</Button>
						<Button type="submit" disabled={loading}>
							{loading ? (
								<Loader2 className="h-4 w-4 animate-spin" />
							) : (
								<Undo2 className="h-4 w-4" />
							)}
							{textGet("cash.refund.submit")}
						</Button>
					</div>
				</div>
			)}
		</Form>
	);
}
