import { useText } from "@pengi/shared";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";
import { ArrowLeftRight } from "lucide-react";
import { useState } from "react";
import {
	type CashMovement,
	createMovement,
	type MovementType,
} from "@/api/cash-service";
import {
	MovementForm,
	type MovementValues,
} from "@/sections/forms/cash/movement-form";

/** Records a manual cash movement in the open session. */
export function MovementDialog({
	open,
	allowedTypes,
	onOpenChange,
	onCreated,
}: {
	open: boolean;
	allowedTypes: MovementType[];
	onOpenChange: (open: boolean) => void;
	onCreated: (movement: CashMovement) => void;
}) {
	const { textGet } = useText();
	const [loading, setLoading] = useState(false);

	async function handleSubmit(values: MovementValues) {
		setLoading(true);
		const res = await createMovement(values);
		setLoading(false);
		if (res.success) {
			onOpenChange(false);
			onCreated(res.data);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[480px]">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
							<ArrowLeftRight aria-hidden className="h-4 w-4" />
						</span>
						{textGet("cash.movement.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet("cash.movement.description")}
					</DialogDescription>
				</DialogHeader>
				{open && (
					<MovementForm
						allowedTypes={allowedTypes}
						loading={loading}
						onSubmit={handleSubmit}
						onCancel={() => onOpenChange(false)}
					/>
				)}
			</DialogContent>
		</Dialog>
	);
}
