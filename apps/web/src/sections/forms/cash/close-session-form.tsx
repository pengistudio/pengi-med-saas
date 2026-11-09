import { useText } from "@pengi/shared";
import { Button, Form, FormInput, FormTextArea } from "@pengi/ui";
import { Banknote, Loader2, Lock, TriangleAlert } from "lucide-react";
import { z } from "zod";

const countedSchema = z.object({
	counted_cash: z
		.number({ error: "cash.form.error.amount" })
		.min(0, "cash.form.error.amount"),
	note: z.string().trim().max(1000, "cash.form.error.too_long").optional(),
});

// Once the count did not match, the note becomes required.
const withNoteSchema = countedSchema.extend({
	note: z
		.string()
		.trim()
		.min(1, "cash.close.error.note_required")
		.max(1000, "cash.form.error.too_long"),
});

export type CloseSessionValues = z.infer<typeof countedSchema>;

interface CloseSessionFormProps {
	/** The count did not match: ask for the note before closing. */
	noteRequired: boolean;
	defaultValues?: Partial<CloseSessionValues>;
	loading?: boolean;
	onSubmit: (values: CloseSessionValues) => void;
	onCancel: () => void;
}

/**
 * The blind close: only the counted cash (and a note). It never receives the
 * opening float, the cash totals nor the expected amount.
 */
export function CloseSessionForm({
	noteRequired,
	defaultValues,
	loading,
	onSubmit,
	onCancel,
}: CloseSessionFormProps) {
	const { textGet } = useText();

	return (
		<Form
			// A new schema needs a new form; the counted amount carries over.
			key={noteRequired ? "with-note" : "count"}
			schema={noteRequired ? withNoteSchema : countedSchema}
			defaultValues={{
				counted_cash: defaultValues?.counted_cash,
				note: defaultValues?.note ?? "",
			}}
			onSubmit={onSubmit}
		>
			{(field) => (
				<div className="space-y-4">
					{/* The count is the whole point of the dialog: large, focused. */}
					<div className="rounded-lg bg-muted/50 p-3 sm:p-4">
						<FormInput
							field={field}
							name="counted_cash"
							type="number"
							step="0.01"
							min={0}
							inputMode="decimal"
							label={textGet("cash.close.counted")}
							description={textGet("cash.close.counted.description")}
							startAddon={<Banknote aria-hidden className="h-5 w-5" />}
							containerClassName="h-12 bg-background"
							className="text-xl font-semibold tabular-nums md:text-xl"
							autoFocus={!noteRequired}
						/>
					</div>
					{noteRequired && (
						<p
							role="alert"
							className="flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-800 dark:text-amber-300"
						>
							<TriangleAlert aria-hidden className="mt-0.5 h-4 w-4 shrink-0" />
							{textGet("cash.close.note_required")}
						</p>
					)}
					<FormTextArea
						field={field}
						name="note"
						label={textGet("cash.close.note")}
						placeholder={textGet("cash.close.note.placeholder")}
						maxLength={1000}
						isOptional={!noteRequired}
						autoFocus={noteRequired}
					/>
					<div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
						<Button type="button" variant="outline" onClick={onCancel}>
							{textGet("common.cancel")}
						</Button>
						<Button type="submit" size="lg" disabled={loading}>
							{loading ? (
								<Loader2 className="h-4 w-4 animate-spin" />
							) : (
								<Lock className="h-4 w-4" />
							)}
							{textGet("cash.close.submit")}
						</Button>
					</div>
				</div>
			)}
		</Form>
	);
}
