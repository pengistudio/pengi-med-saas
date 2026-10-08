import { parseDateOnly, useText } from "@pengi/shared";
import { Button, Checkbox, Spinner } from "@pengi/ui";
import { CalendarOff, Pencil, Plus, Trash2 } from "lucide-react";
import React from "react";
import {
	type AffectedAppointment,
	type BlockPayload,
	type BlockScope,
	createBlock,
	deleteBlock,
	getBlocks,
	type ScheduleBlock,
	updateBlock,
} from "@/api/agenda-service";
import { ConfirmActionDialog } from "@/components/features/exam-orders/confirm-action-dialog";
import { FormDialog } from "@/components/features/exam-orders/exam-catalog-parts";
import { ScheduleBlockForm } from "@/sections/forms/agenda/schedule-block-form";
import { AffectedAppointmentsDialog } from "./affected-appointments-dialog";

/** Lists include past blocks from this date when "show past" is on. */
const PAST_FROM = "2000-01-01";

type Editing = { mode: "create" } | { mode: "edit"; block: ScheduleBlock };

/**
 * Days or hours off of a doctor (own or any, by `scope`) or of the whole
 * clinic. Saving one over existing appointments is allowed; the affected
 * appointments are listed afterwards.
 */
export function ScheduleBlocksList({
	scope,
	canEdit = true,
}: {
	scope: BlockScope;
	canEdit?: boolean;
}) {
	const { textGet, formatDate } = useText();
	const [blocks, setBlocks] = React.useState<ScheduleBlock[]>([]);
	const [loading, setLoading] = React.useState(true);
	const [showPast, setShowPast] = React.useState(false);
	const idPrefix = React.useId();
	const [editing, setEditing] = React.useState<Editing | null>(null);
	const [saving, setSaving] = React.useState(false);
	const [toDelete, setToDelete] = React.useState<ScheduleBlock | null>(null);
	const [affected, setAffected] = React.useState<AffectedAppointment[] | null>(
		null,
	);
	// Only the latest load writes (scope or "show past" may change mid-flight).
	const loadSeq = React.useRef(0);
	const reload = React.useCallback(() => {
		const seq = ++loadSeq.current;
		setLoading(true);
		getBlocks(scope, showPast ? { from: PAST_FROM } : {}).then((res) => {
			if (seq !== loadSeq.current) return;
			setBlocks(res.success && res.data ? res.data : []);
			setLoading(false);
		});
	}, [scope, showPast]);

	React.useEffect(() => {
		reload();
	}, [reload]);

	async function handleSave(payload: BlockPayload) {
		if (!editing) return;
		setSaving(true);
		const res =
			editing.mode === "edit"
				? await updateBlock(scope, editing.block.ID, payload)
				: await createBlock(scope, payload);
		setSaving(false);
		if (res.success) {
			setEditing(null);
			const list = res.data?.affected_appointments ?? [];
			if (list.length > 0) setAffected(list);
			reload();
		}
	}

	async function handleDelete() {
		if (!toDelete) return;
		const res = await deleteBlock(scope, toDelete.ID);
		setToDelete(null);
		if (res.success) reload();
	}

	function dateLabel(block: ScheduleBlock) {
		const start = formatDate(parseDateOnly(block.start_date), "medium");
		if (block.start_date === block.end_date) return start;
		return textGet("agenda.blocks.date_range", {
			start,
			end: formatDate(parseDateOnly(block.end_date), "medium"),
		});
	}

	return (
		<div className="grid gap-3">
			<div className="flex flex-wrap items-center justify-between gap-3">
				<div className="flex items-center gap-2 text-sm text-muted-foreground">
					<Checkbox
						id={`${idPrefix}-past`}
						checked={showPast}
						onCheckedChange={(checked) => setShowPast(!!checked)}
					/>
					<label htmlFor={`${idPrefix}-past`} className="cursor-pointer">
						{textGet("agenda.blocks.show_past")}
					</label>
				</div>
				{canEdit && (
					<Button
						type="button"
						variant="outline"
						onClick={() => setEditing({ mode: "create" })}
					>
						<Plus className="mr-2 h-4 w-4" />
						{textGet("agenda.blocks.new")}
					</Button>
				)}
			</div>

			{loading ? (
				<div className="flex justify-center py-6">
					<Spinner />
				</div>
			) : blocks.length === 0 ? (
				<p className="flex items-center gap-2 rounded-lg border border-dashed px-3 py-6 text-sm text-muted-foreground justify-center">
					<CalendarOff className="h-4 w-4" />
					{textGet("agenda.blocks.empty")}
				</p>
			) : (
				<ul className="divide-y rounded-lg border">
					{blocks.map((block) => (
						<li
							key={block.ID}
							className="flex items-center justify-between gap-3 px-3 py-2"
						>
							<div className="grid min-w-0 gap-0.5 text-sm">
								<span className="font-medium">{dateLabel(block)}</span>
								<span className="truncate text-muted-foreground">
									{block.start_time
										? `${block.start_time}–${block.end_time}`
										: textGet("agenda.blocks.full_day")}
									{" · "}
									{block.reason || textGet("agenda.blocks.no_reason")}
								</span>
							</div>
							{canEdit && (
								<div className="flex shrink-0 gap-1">
									<Button
										type="button"
										variant="ghost"
										size="icon"
										aria-label={textGet("agenda.blocks.edit")}
										onClick={() => setEditing({ mode: "edit", block })}
									>
										<Pencil className="h-4 w-4" />
									</Button>
									<Button
										type="button"
										variant="ghost"
										size="icon"
										aria-label={textGet("agenda.blocks.delete")}
										onClick={() => setToDelete(block)}
									>
										<Trash2 className="h-4 w-4" />
									</Button>
								</div>
							)}
						</li>
					))}
				</ul>
			)}

			<FormDialog
				open={editing !== null}
				busy={saving}
				onClose={() => setEditing(null)}
				title={
					editing?.mode === "edit"
						? textGet("agenda.blocks.edit")
						: textGet("agenda.blocks.new")
				}
			>
				{editing && (
					<ScheduleBlockForm
						block={editing.mode === "edit" ? editing.block : undefined}
						loading={saving}
						onSubmit={handleSave}
					/>
				)}
			</FormDialog>

			<ConfirmActionDialog
				open={toDelete !== null}
				onOpenChange={(open) => {
					if (!open) setToDelete(null);
				}}
				title={textGet("agenda.blocks.delete.title")}
				description={textGet("agenda.blocks.delete.description")}
				onConfirm={handleDelete}
			/>

			<AffectedAppointmentsDialog
				appointments={affected}
				onClose={() => setAffected(null)}
			/>
		</div>
	);
}
