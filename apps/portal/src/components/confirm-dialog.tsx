import { useText } from "@pengi/shared";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
	Spinner,
} from "@pengi/ui";
import React from "react";

/**
 * Asks before an action the patient cannot undo (cancelling). `onConfirm`
 * resolves to whether it worked; the dialog closes only then.
 */
export function ConfirmDialog({
	open,
	onOpenChange,
	title,
	description,
	confirmLabel,
	onConfirm,
}: {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	title: string;
	description: string;
	confirmLabel: string;
	onConfirm: () => Promise<boolean>;
}) {
	const { textGet } = useText();
	const [busy, setBusy] = React.useState(false);

	const confirm = async () => {
		setBusy(true);
		const ok = await onConfirm();
		setBusy(false);
		if (ok) onOpenChange(false);
	};

	return (
		<AlertDialog
			open={open}
			onOpenChange={(next) => !busy && onOpenChange(next)}
		>
			<AlertDialogContent className="rounded-3xl">
				<AlertDialogHeader>
					<AlertDialogTitle>{title}</AlertDialogTitle>
					<AlertDialogDescription>{description}</AlertDialogDescription>
				</AlertDialogHeader>
				<AlertDialogFooter>
					<AlertDialogCancel className="rounded-full" disabled={busy}>
						{textGet("portal.booking.keep")}
					</AlertDialogCancel>
					<AlertDialogAction
						variant="destructive"
						className="rounded-full"
						disabled={busy}
						onClick={confirm}
					>
						{busy && <Spinner />}
						{confirmLabel}
					</AlertDialogAction>
				</AlertDialogFooter>
			</AlertDialogContent>
		</AlertDialog>
	);
}
