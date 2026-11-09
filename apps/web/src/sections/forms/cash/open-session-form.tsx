import { useText } from "@pengi/shared";
import { Button, Form, FormInput } from "@pengi/ui";
import { DollarSign, Loader2, LockOpen } from "lucide-react";
import { z } from "zod";

const openSessionSchema = z.object({
	opening_float: z
		.number({ error: "cash.form.error.amount" })
		.min(0, "cash.form.error.amount"),
});

export type OpenSessionValues = z.infer<typeof openSessionSchema>;

/**
 * Opens the cash desk with the cash already in the drawer (opening float):
 * one amount and the action beside it, so it reads as the page's next step.
 */
export function OpenSessionForm({
	loading,
	onSubmit,
}: {
	loading?: boolean;
	onSubmit: (values: OpenSessionValues) => void;
}) {
	const { textGet } = useText();

	return (
		<Form
			schema={openSessionSchema}
			defaultValues={{ opening_float: 0 }}
			onSubmit={onSubmit}
		>
			{(field) => (
				<div className="flex flex-col gap-3 sm:flex-row sm:items-start">
					<div className="sm:w-64">
						<FormInput
							field={field}
							name="opening_float"
							type="number"
							step="0.01"
							min={0}
							inputMode="decimal"
							label={textGet("cash.open.float")}
							description={textGet("cash.open.float.description")}
							startAddon={<DollarSign aria-hidden className="h-4 w-4" />}
							containerClassName="h-10"
							className="text-base tabular-nums"
						/>
					</div>
					<Button
						type="submit"
						size="lg"
						disabled={loading}
						className="h-10 px-4 sm:mt-[1.375rem]"
					>
						{loading ? (
							<Loader2 className="h-4 w-4 animate-spin" />
						) : (
							<LockOpen className="h-4 w-4" />
						)}
						{textGet("cash.open.submit")}
					</Button>
				</div>
			)}
		</Form>
	);
}
