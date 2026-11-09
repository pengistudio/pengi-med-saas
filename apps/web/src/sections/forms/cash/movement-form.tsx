import { useText } from "@pengi/shared";
import {
	Button,
	Form,
	FormInput,
	FormRow,
	FormSelect,
	FormTextArea,
} from "@pengi/ui";
import { ArrowDownLeft, ArrowUpRight, DollarSign, Loader2 } from "lucide-react";
import { useEffect } from "react";
import type { UseFormReturn } from "react-hook-form";
import { z } from "zod";
import {
	MOVEMENT_CATEGORIES,
	type MovementCategory,
	type MovementType,
} from "@/api/cash-service";
import {
	movementCategoryLabel,
	movementTypeLabel,
} from "@/components/features/cash/cash-labels";
import { cn } from "@/lib/utils";

const movementSchema = z
	.object({
		type: z.enum(["in", "out"], { error: "cash.form.error.required" }),
		category: z.enum(["expense", "withdrawal", "deposit", "adjustment"], {
			error: "cash.form.error.required",
		}),
		amount: z
			.number({ error: "cash.form.error.amount" })
			.positive("cash.form.error.amount"),
		reason: z
			.string()
			.trim()
			.min(1, "cash.form.error.reason_required")
			.max(1000, "cash.form.error.too_long"),
	})
	.refine((v) => MOVEMENT_CATEGORIES[v.type].includes(v.category), {
		path: ["category"],
		message: "cash.form.error.required",
	});

export type MovementValues = z.infer<typeof movementSchema>;

interface MovementFormProps {
	/** Types the user may record: in needs MANAGE_CASH_SESSION, out VOID_PAYMENT. */
	allowedTypes: MovementType[];
	loading?: boolean;
	onSubmit: (values: MovementValues) => void;
	onCancel: () => void;
}

/** Cash put into or taken out of the drawer that is not a payment. */
export function MovementForm({
	allowedTypes,
	loading,
	onSubmit,
	onCancel,
}: MovementFormProps) {
	const initialType = allowedTypes[0] ?? "in";
	return (
		<Form
			schema={movementSchema}
			defaultValues={{
				type: initialType,
				category: MOVEMENT_CATEGORIES[initialType][0],
				reason: "",
			}}
			onSubmit={onSubmit}
		>
			{(field) => (
				<MovementFields
					field={field}
					allowedTypes={allowedTypes}
					loading={loading}
					onCancel={onCancel}
				/>
			)}
		</Form>
	);
}

function MovementFields({
	field,
	allowedTypes,
	loading,
	onCancel,
}: {
	field: UseFormReturn<z.input<typeof movementSchema>, unknown, MovementValues>;
	allowedTypes: MovementType[];
	loading?: boolean;
	onCancel: () => void;
}) {
	const { textGet } = useText();
	const type = field.watch("type") as MovementType;
	const category = field.watch("category") as MovementCategory;
	const categories = MOVEMENT_CATEGORIES[type] ?? [];

	// A category that the new type doesn't allow falls back to its first one.
	useEffect(() => {
		if (!categories.includes(category) && categories[0]) {
			field.setValue("category", categories[0]);
		}
	}, [categories, category, field]);

	return (
		<div className="space-y-4">
			<div className="space-y-2">
				<p id="cash-movement-type" className="text-sm leading-none font-medium">
					{textGet("cash.movement.type")}
				</p>
				<div
					role="radiogroup"
					aria-labelledby="cash-movement-type"
					className="grid grid-cols-2 gap-2"
				>
					{allowedTypes.map((t) => {
						const Icon = t === "in" ? ArrowDownLeft : ArrowUpRight;
						const selected = type === t;
						return (
							<label
								key={t}
								className={cn(
									"flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2.5 text-left text-sm font-medium outline-none transition-colors duration-(--motion-fast) has-focus-visible:ring-3 has-focus-visible:ring-ring/50",
									selected
										? t === "in"
											? "border-emerald-600/40 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300"
											: "border-destructive/40 bg-destructive/10 text-destructive"
										: "hover:bg-muted",
								)}
							>
								<input
									type="radio"
									name="cash-movement-type"
									value={t}
									checked={selected}
									onChange={() =>
										field.setValue("type", t, { shouldValidate: true })
									}
									className="sr-only"
								/>
								<Icon aria-hidden className="h-4 w-4 shrink-0" />
								{movementTypeLabel(textGet, t)}
							</label>
						);
					})}
				</div>
			</div>
			<FormRow>
				<FormSelect
					field={field}
					name="category"
					label={textGet("cash.movement.category")}
					options={categories.map((c) => ({
						value: c,
						label: movementCategoryLabel(textGet, c),
					}))}
				/>
				<FormInput
					field={field}
					name="amount"
					type="number"
					step="0.01"
					min={0}
					inputMode="decimal"
					label={textGet("cash.movement.amount")}
					startAddon={<DollarSign aria-hidden className="h-4 w-4" />}
					className="tabular-nums"
				/>
			</FormRow>
			<FormTextArea
				field={field}
				name="reason"
				label={textGet("cash.movement.reason")}
				placeholder={textGet("cash.movement.reason.placeholder")}
				maxLength={1000}
			/>
			<div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
				<Button type="button" variant="outline" onClick={onCancel}>
					{textGet("common.cancel")}
				</Button>
				<Button type="submit" disabled={loading}>
					{loading && <Loader2 className="h-4 w-4 animate-spin" />}
					{textGet("cash.movement.submit")}
				</Button>
			</div>
		</div>
	);
}
