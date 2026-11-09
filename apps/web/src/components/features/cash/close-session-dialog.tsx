import { useText } from "@pengi/shared";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";
import { Lock } from "lucide-react";
import { useState } from "react";
import {
	CASH_NOTE_REQUIRED,
	type CashSessionDetail,
	closeSession,
} from "@/api/cash-service";
import {
	CloseSessionForm,
	type CloseSessionValues,
} from "@/sections/forms/cash/close-session-form";

/**
 * Blind close of the open session: asks only for the counted cash. When the
 * API answers that the count differs (E-CASH-005) it asks for the note and
 * resends; the expected amount is shown only after the close succeeds.
 */
export function CloseSessionDialog({
	sessionId,
	open,
	onOpenChange,
	onClosed,
}: {
	sessionId: number;
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onClosed: (session: CashSessionDetail) => void;
}) {
	const { textGet } = useText();
	const [loading, setLoading] = useState(false);
	const [noteRequired, setNoteRequired] = useState(false);
	const [values, setValues] = useState<Partial<CloseSessionValues>>({});

	function handleOpenChange(next: boolean) {
		if (!next) {
			setNoteRequired(false);
			setValues({});
		}
		onOpenChange(next);
	}

	async function handleSubmit(submitted: CloseSessionValues) {
		setValues(submitted);
		setLoading(true);
		const res = await closeSession(sessionId, {
			counted_cash: submitted.counted_cash,
			note: submitted.note?.trim() || undefined,
		});
		setLoading(false);
		if (res.success) {
			handleOpenChange(false);
			onClosed(res.data);
			return;
		}
		if (res.data?.error_code === CASH_NOTE_REQUIRED) setNoteRequired(true);
	}

	return (
		<Dialog open={open} onOpenChange={handleOpenChange}>
			<DialogContent className="sm:max-w-[440px]">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
							<Lock aria-hidden className="h-4 w-4" />
						</span>
						{textGet("cash.close.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet("cash.close.description")}
					</DialogDescription>
				</DialogHeader>
				<CloseSessionForm
					noteRequired={noteRequired}
					defaultValues={values}
					loading={loading}
					onSubmit={handleSubmit}
					onCancel={() => handleOpenChange(false)}
				/>
			</DialogContent>
		</Dialog>
	);
}
