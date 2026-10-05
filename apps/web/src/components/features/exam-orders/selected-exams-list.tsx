import { useText } from "@pengi/shared";
import { Badge, Button, Input } from "@pengi/ui";
import { Lock, X } from "lucide-react";
import { type DraftExamItem, EXAM_CATEGORY_KEYS } from "@/lib/exam-orders";

interface SelectedExamsListProps {
	value: DraftExamItem[];
	onChange: (drafts: DraftExamItem[]) => void;
}

/**
 * The exams chosen for the order, each with editable indications (prefilled
 * with the catalog's). Locked exams (the order already has results) are
 * read-only and can't be removed.
 */
export function SelectedExamsList({ value, onChange }: SelectedExamsListProps) {
	const { textGet } = useText();

	if (value.length === 0) {
		return (
			<p className="rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground">
				{textGet("clinical.exam_orders.selected.empty")}
			</p>
		);
	}

	const update = (key: string, indications: string) =>
		onChange(value.map((d) => (d.key === key ? { ...d, indications } : d)));

	return (
		<ul className="divide-y rounded-md border">
			{value.map((draft) => (
				<li key={draft.key} className="space-y-2 p-3">
					<div className="flex items-start gap-2">
						<div className="min-w-0 flex-1">
							<p className="text-sm font-medium">{draft.name}</p>
							<div className="mt-1 flex flex-wrap gap-1">
								<Badge variant="secondary">
									{textGet(EXAM_CATEGORY_KEYS[draft.category])}
								</Badge>
								{draft.subgroup && (
									<Badge variant="outline">{draft.subgroup}</Badge>
								)}
							</div>
						</div>
						{draft.locked ? (
							<Lock
								className="mt-1 h-4 w-4 shrink-0 text-muted-foreground"
								aria-label={textGet("clinical.exam_orders.selected.locked")}
							/>
						) : (
							<Button
								type="button"
								variant="ghost"
								size="icon"
								aria-label={textGet("clinical.exam_orders.selected.remove")}
								onClick={() =>
									onChange(value.filter((d) => d.key !== draft.key))
								}
							>
								<X className="h-4 w-4" />
							</Button>
						)}
					</div>
					<Input
						value={draft.indications}
						disabled={draft.locked}
						onChange={(e) => update(draft.key, e.target.value)}
						placeholder={textGet(
							"clinical.exam_orders.field.indications.placeholder",
						)}
						aria-label={textGet("clinical.exam_orders.field.indications")}
					/>
				</li>
			))}
		</ul>
	);
}
