import { useText } from "@pengi/shared";
import { Button, Form, FormInput, FormSelect } from "@pengi/ui";
import { CircleCheck, DollarSign, Loader2, Plus, Trash } from "lucide-react";
import { useMemo } from "react";
import { type UseFormReturn, useFieldArray } from "react-hook-form";
import { z } from "zod";
import { PAYMENT_METHODS } from "@/api/cash-service";
import { MethodIcon } from "@/components/features/cash/cash-figures";
import { paymentMethodLabel } from "@/components/features/cash/cash-labels";
import { cn } from "@/lib/utils";

const toCents = (v: number) => Math.round((Number(v) || 0) * 100);

const lineSchema = z.object({
	method: z.enum(PAYMENT_METHODS, { error: "cash.form.error.required" }),
	amount: z
		.number({ error: "cash.form.error.amount" })
		.positive("cash.form.error.amount"),
	reference: z.string().trim().max(100, "cash.form.error.too_long").optional(),
});

/** The lines may not add up to more than the invoice's balance. */
function paymentSchema(balance: number) {
	return z.object({
		lines: z
			.array(lineSchema)
			.min(1, "cash.payment.error.no_lines")
			.refine(
				(lines) =>
					lines.reduce((sum, l) => sum + toCents(l.amount), 0) <=
					toCents(balance),
				"cash.payment.error.exceeds_balance",
			),
	});
}

export type PaymentValues = z.infer<ReturnType<typeof paymentSchema>>;
type PaymentInput = z.input<ReturnType<typeof paymentSchema>>;

interface PaymentFormProps {
	/** What is left to collect on the invoice. */
	balance: number;
	loading?: boolean;
	cancelLabel?: string;
	onSubmit: (values: PaymentValues) => void;
	onCancel: () => void;
}

/**
 * Charges an invoice with one or more lines (mixed methods). A line's
 * reference is asked only for methods that have one (not cash); the dialog
 * drops it from cash lines on submit.
 */
export function PaymentForm({
	balance,
	loading,
	cancelLabel,
	onSubmit,
	onCancel,
}: PaymentFormProps) {
	const schema = useMemo(() => paymentSchema(balance), [balance]);

	return (
		<Form
			schema={schema}
			defaultValues={{
				lines: [{ method: "cash", amount: balance, reference: "" }],
			}}
			onSubmit={onSubmit}
		>
			{(field) => (
				<PaymentLines
					field={field}
					balance={balance}
					loading={loading}
					cancelLabel={cancelLabel}
					onCancel={onCancel}
				/>
			)}
		</Form>
	);
}

/**
 * The balance and what is still left, live: a bar for how much of the
 * balance the lines cover, and the remaining amount (or "covered", or the
 * excess as an error).
 */
function BalanceStatus({
	balanceCents,
	enteredCents,
}: {
	balanceCents: number;
	enteredCents: number;
}) {
	const { textGet, formatMoney } = useText();
	const remainingCents = balanceCents - enteredCents;
	const exceeds = remainingCents < 0;
	const covered = remainingCents === 0;
	const ratio =
		balanceCents > 0
			? Math.min(Math.max(enteredCents / balanceCents, 0), 1)
			: 0;

	return (
		<div className="space-y-3 rounded-lg bg-muted/50 p-3 sm:p-4">
			<div className="flex flex-wrap items-end justify-between gap-x-6 gap-y-2">
				<div>
					<p className="text-xs text-muted-foreground">
						{textGet("cash.payment.balance")}
					</p>
					<p className="text-lg font-semibold tabular-nums">
						{formatMoney(balanceCents / 100)}
					</p>
				</div>
				<div className="text-right">
					<p className="text-xs text-muted-foreground">
						{textGet("cash.payment.entered")}
					</p>
					<p className="text-lg font-semibold tabular-nums">
						{formatMoney(enteredCents / 100)}
					</p>
				</div>
			</div>
			<div
				role="progressbar"
				aria-label={textGet("cash.payment.progress")}
				aria-valuemin={0}
				aria-valuemax={100}
				aria-valuenow={Math.round(ratio * 100)}
				className="h-2 overflow-hidden rounded-full bg-foreground/10"
			>
				<div
					className={cn(
						"h-full rounded-full transition-[width] duration-(--motion-base) ease-out-soft motion-reduce:transition-none",
						exceeds
							? "bg-destructive"
							: covered
								? "bg-emerald-500"
								: "bg-primary",
					)}
					style={{ width: `${exceeds ? 100 : ratio * 100}%` }}
				/>
			</div>
			{exceeds ? (
				<p role="alert" className="text-sm font-medium text-destructive">
					{textGet("cash.payment.error.exceeds_balance")}
				</p>
			) : covered ? (
				<p className="flex items-center gap-1.5 text-sm font-medium text-emerald-700 dark:text-emerald-400">
					<CircleCheck aria-hidden className="h-4 w-4" />
					{textGet("cash.payment.covered")}
				</p>
			) : (
				<p className="flex items-baseline justify-between gap-4 text-sm font-medium">
					<span>{textGet("cash.payment.remaining")}</span>
					<span className="text-base text-amber-700 tabular-nums dark:text-amber-400">
						{formatMoney(remainingCents / 100)}
					</span>
				</p>
			)}
		</div>
	);
}

function PaymentLines({
	field,
	balance,
	loading,
	cancelLabel,
	onCancel,
}: {
	field: UseFormReturn<PaymentInput, unknown, PaymentValues>;
	balance: number;
	loading?: boolean;
	cancelLabel?: string;
	onCancel: () => void;
}) {
	const { textGet } = useText();
	const { fields, append, remove } = useFieldArray({
		control: field.control,
		name: "lines",
	});
	const lines = field.watch("lines") ?? [];
	const entered = lines.reduce(
		(sum, l) => sum + toCents(l.amount as number),
		0,
	);
	const balanceCents = toCents(balance);
	const remainingCents = balanceCents - entered;
	// Shown live, before submitting; the schema refine is the same check.
	const exceeds = remainingCents < 0;
	const methodOptions = useMemo(
		() =>
			PAYMENT_METHODS.map((m) => ({
				value: m,
				label: (
					<span className="flex items-center gap-2">
						<MethodIcon method={m} className="text-muted-foreground" />
						{paymentMethodLabel(textGet, m)}
					</span>
				),
			})),
		[textGet],
	);
	const removable = fields.length > 1;

	return (
		<div className="space-y-4">
			<BalanceStatus balanceCents={balanceCents} enteredCents={entered} />

			<div className="space-y-3">
				{fields.map((line, index) => {
					const hasReference = lines[index]?.method !== "cash";
					return (
						<div key={line.id} className="relative rounded-lg border p-3">
							<div className="grid gap-3 sm:grid-cols-2">
								<FormSelect
									field={field}
									name={`lines.${index}.method` as const}
									label={textGet("cash.payment.method")}
									options={methodOptions}
								/>
								<FormInput
									field={field}
									name={`lines.${index}.amount` as const}
									type="number"
									step="0.01"
									min={0}
									inputMode="decimal"
									label={textGet("cash.payment.amount")}
									startAddon={<DollarSign aria-hidden className="h-4 w-4" />}
									className="tabular-nums"
								/>
								{hasReference && (
									<div className="sm:col-span-2">
										<FormInput
											field={field}
											name={`lines.${index}.reference` as const}
											label={textGet("cash.payment.reference")}
											placeholder={textGet(
												"cash.payment.reference.placeholder",
											)}
											isOptional
										/>
									</div>
								)}
							</div>
							{removable && (
								<Button
									type="button"
									variant="ghost"
									size="icon-sm"
									className="absolute top-1.5 right-1.5 text-muted-foreground"
									onClick={() => remove(index)}
									aria-label={textGet("cash.payment.remove_line")}
								>
									<Trash className="h-4 w-4" />
								</Button>
							)}
						</div>
					);
				})}

				<Button
					type="button"
					variant="outline"
					size="sm"
					disabled={remainingCents <= 0}
					onClick={() =>
						append({
							method: "transfer",
							amount: Math.max(remainingCents, 0) / 100,
							reference: "",
						})
					}
				>
					<Plus className="h-4 w-4" />
					{textGet("cash.payment.add_line")}
				</Button>
			</div>

			<div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
				<Button type="button" variant="outline" onClick={onCancel}>
					{cancelLabel ?? textGet("common.cancel")}
				</Button>
				<Button type="submit" size="lg" disabled={loading || exceeds}>
					{loading && <Loader2 className="h-4 w-4 animate-spin" />}
					{textGet("cash.payment.submit")}
				</Button>
			</div>
		</div>
	);
}
