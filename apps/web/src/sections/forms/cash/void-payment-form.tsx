import { useText } from "@pengi/shared";
import { Button, Form, FormTextArea } from "@pengi/ui";
import { Ban, Loader2 } from "lucide-react";
import { z } from "zod";

const voidSchema = z.object({
	reason: z
		.string()
		.trim()
		.min(1, "cash.form.error.reason_required")
		.max(1000, "cash.form.error.too_long"),
});

export type VoidPaymentValues = z.infer<typeof voidSchema>;

/** The reason a payment of the open session is voided. */
export function VoidPaymentForm({
	loading,
	onSubmit,
	onCancel,
}: {
	loading?: boolean;
	onSubmit: (values: VoidPaymentValues) => void;
	onCancel: () => void;
}) {
	const { textGet } = useText();

	return (
		<Form
			schema={voidSchema}
			defaultValues={{ reason: "" }}
			onSubmit={onSubmit}
		>
			{(field) => (
				<div className="space-y-4">
					<FormTextArea
						field={field}
						name="reason"
						label={textGet("cash.void.reason")}
						placeholder={textGet("cash.void.reason.placeholder")}
						maxLength={1000}
						autoFocus
					/>
					<div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
						<Button type="button" variant="outline" onClick={onCancel}>
							{textGet("common.cancel")}
						</Button>
						<Button type="submit" variant="destructive" disabled={loading}>
							{loading ? (
								<Loader2 className="h-4 w-4 animate-spin" />
							) : (
								<Ban className="h-4 w-4" />
							)}
							{textGet("cash.void.submit")}
						</Button>
					</div>
				</div>
			)}
		</Form>
	);
}
