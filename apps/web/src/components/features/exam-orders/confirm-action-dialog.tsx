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
	AlertDialogTrigger,
} from "@pengi/ui";
import type React from "react";

interface ConfirmActionDialogProps {
	/** The button that opens it; omit to control it with `open`. */
	trigger?: React.ReactElement;
	open?: boolean;
	onOpenChange?: (open: boolean) => void;
	title: React.ReactNode;
	description: React.ReactNode;
	onConfirm: () => void;
}

/** "Are you sure?" before an action of the exam orders screens. */
export function ConfirmActionDialog({
	trigger,
	open,
	onOpenChange,
	title,
	description,
	onConfirm,
}: ConfirmActionDialogProps) {
	const { textGet } = useText();
	return (
		<AlertDialog open={open} onOpenChange={onOpenChange}>
			{trigger && <AlertDialogTrigger render={trigger} />}
			<AlertDialogContent>
				<AlertDialogHeader>
					<AlertDialogTitle>{title}</AlertDialogTitle>
					<AlertDialogDescription>{description}</AlertDialogDescription>
				</AlertDialogHeader>
				<AlertDialogFooter>
					<AlertDialogCancel>{textGet("form.cancel")}</AlertDialogCancel>
					<AlertDialogAction onClick={onConfirm}>
						{textGet("form.continue")}
					</AlertDialogAction>
				</AlertDialogFooter>
			</AlertDialogContent>
		</AlertDialog>
	);
}
