import { useText } from "@pengi/shared";
import {
	Button,
	Checkbox,
	Input,
	Popover,
	PopoverContent,
	PopoverTrigger,
	Spinner,
} from "@pengi/ui";
import { Copy, Loader2, Plus, Save, Trash2 } from "lucide-react";
import React from "react";
import {
	type DoctorSchedule,
	getSchedule,
	replaceSchedule,
	type ScheduleScope,
	type ScheduleSlot,
} from "@/api/agenda-service";
import {
	addMinutes,
	endFromInput,
	endToInput,
	validateWeek,
	WEEKDAY_KEYS,
	WEEKDAY_ORDER,
	type WeekDraft,
} from "./agenda-utils";

function emptyWeek(): WeekDraft {
	return { 0: [], 1: [], 2: [], 3: [], 4: [], 5: [], 6: [] };
}

function toDraft(rows: DoctorSchedule[]): WeekDraft {
	const week = emptyWeek();
	for (const row of rows) {
		week[row.weekday]?.push({
			start_time: row.start_time,
			end_time: row.end_time,
		});
	}
	return week;
}

function toSlots(week: WeekDraft): ScheduleSlot[] {
	return WEEKDAY_ORDER.flatMap((weekday) =>
		week[weekday].map((r) => ({ weekday, ...r })),
	);
}

/** Copies one day's ranges onto the days ticked in the popover. */
function CopyDayButton({
	from,
	onCopy,
}: {
	from: number;
	onCopy: (targets: number[]) => void;
}) {
	const { textGet } = useText();
	const [open, setOpen] = React.useState(false);
	const [targets, setTargets] = React.useState<number[]>([]);
	const idPrefix = React.useId();
	return (
		<Popover
			open={open}
			onOpenChange={(next) => {
				setOpen(next);
				if (next) setTargets([]);
			}}
		>
			<PopoverTrigger
				render={
					<Button
						type="button"
						variant="ghost"
						size="icon"
						aria-label={textGet("agenda.schedule.copy")}
						title={textGet("agenda.schedule.copy")}
					>
						<Copy className="h-4 w-4" />
					</Button>
				}
			/>
			<PopoverContent align="end" className="w-56">
				<p className="text-sm font-medium">
					{textGet("agenda.schedule.copy_to")}
				</p>
				<div className="grid gap-1.5">
					{WEEKDAY_ORDER.filter((d) => d !== from).map((day) => (
						<div key={day} className="flex items-center gap-2 text-sm">
							<Checkbox
								id={`${idPrefix}-${day}`}
								checked={targets.includes(day)}
								onCheckedChange={(checked) =>
									setTargets((current) =>
										checked
											? [...current, day]
											: current.filter((d) => d !== day),
									)
								}
							/>
							<label htmlFor={`${idPrefix}-${day}`} className="cursor-pointer">
								{textGet(WEEKDAY_KEYS[day])}
							</label>
						</div>
					))}
				</div>
				<Button
					type="button"
					size="sm"
					disabled={targets.length === 0}
					onClick={() => {
						onCopy(targets);
						setOpen(false);
					}}
				>
					{textGet("agenda.schedule.copy_apply")}
				</Button>
			</PopoverContent>
		</Popover>
	);
}

/**
 * A doctor's weekly working hours: ranges per weekday (several allowed),
 * copied between days and saved as a whole week. Without ranges the doctor
 * has no schedule and the agenda shows no warnings or shading for them.
 */
export function WeeklyScheduleEditor({
	scope,
	canEdit = true,
}: {
	scope: ScheduleScope;
	canEdit?: boolean;
}) {
	const { textGet } = useText();
	const [loading, setLoading] = React.useState(true);
	const [saving, setSaving] = React.useState(false);
	const [saved, setSaved] = React.useState<WeekDraft>(emptyWeek);
	const [week, setWeek] = React.useState<WeekDraft>(emptyWeek);

	React.useEffect(() => {
		let current = true;
		setLoading(true);
		getSchedule(scope).then((res) => {
			if (!current) return;
			const draft = toDraft(res.success && res.data ? res.data : []);
			setSaved(draft);
			setWeek(draft);
			setLoading(false);
		});
		return () => {
			current = false;
		};
	}, [scope]);

	const errors = validateWeek(week);
	const hasErrors = Object.keys(errors).length > 0;
	const dirty =
		JSON.stringify(toSlots(week)) !== JSON.stringify(toSlots(saved));

	const updateDay = (
		day: number,
		change: (ranges: WeekDraft[number]) => WeekDraft[number],
	) => setWeek((w) => ({ ...w, [day]: change(w[day]) }));

	function addRange(day: number) {
		updateDay(day, (ranges) => {
			const last = ranges[ranges.length - 1];
			const start = last ? last.end_time : "08:00";
			if (start === "24:00") return ranges;
			return [
				...ranges,
				{ start_time: start, end_time: addMinutes(start, 4 * 60) },
			];
		});
	}

	function copyDay(from: number, targets: number[]) {
		setWeek((w) => {
			const next = { ...w };
			for (const day of targets) next[day] = w[from].map((r) => ({ ...r }));
			return next;
		});
	}

	async function handleSave() {
		setSaving(true);
		const res = await replaceSchedule(scope, toSlots(week));
		setSaving(false);
		if (res.success && res.data) {
			const draft = toDraft(res.data);
			setSaved(draft);
			setWeek(draft);
		}
	}

	if (loading) {
		return (
			<div className="flex justify-center py-8">
				<Spinner />
			</div>
		);
	}

	return (
		<div className="grid gap-4">
			<div className="divide-y rounded-lg border">
				{WEEKDAY_ORDER.map((day) => {
					const ranges = week[day];
					const error = errors[day];
					return (
						<div
							key={day}
							className="grid gap-2 p-3 sm:grid-cols-[6.5rem_1fr_auto] sm:items-start"
						>
							<p className="pt-2 text-sm font-medium">
								{textGet(WEEKDAY_KEYS[day])}
							</p>
							{/* Ranges of a day sit side by side when there is room. */}
							<div className="flex flex-wrap items-center gap-x-6 gap-y-2">
								{ranges.length === 0 && (
									<p className="py-2 text-sm text-muted-foreground">
										{textGet("agenda.schedule.day_off")}
									</p>
								)}
								{ranges.map((range, index) => (
									<div key={index} className="flex items-center gap-2">
										<Input
											type="time"
											step={300}
											value={range.start_time}
											disabled={!canEdit}
											aria-label={textGet("agenda.schedule.start")}
											className="w-[7.75rem]"
											onChange={(e) =>
												updateDay(day, (rs) =>
													rs.map((r, i) =>
														i === index
															? { ...r, start_time: e.target.value }
															: r,
													),
												)
											}
										/>
										<span className="text-muted-foreground">–</span>
										<Input
											type="time"
											step={300}
											value={endToInput(range.end_time)}
											disabled={!canEdit}
											aria-label={textGet("agenda.schedule.end")}
											className="w-[7.75rem]"
											onChange={(e) =>
												updateDay(day, (rs) =>
													rs.map((r, i) =>
														i === index
															? { ...r, end_time: endFromInput(e.target.value) }
															: r,
													),
												)
											}
										/>
										{canEdit && (
											<Button
												type="button"
												variant="ghost"
												size="icon"
												aria-label={textGet("agenda.schedule.remove_range")}
												onClick={() =>
													updateDay(day, (rs) =>
														rs.filter((_, i) => i !== index),
													)
												}
											>
												<Trash2 className="h-4 w-4" />
											</Button>
										)}
									</div>
								))}
								{error && (
									<p className="basis-full text-sm text-destructive">
										{textGet(error)}
									</p>
								)}
							</div>
							{canEdit && (
								<div className="flex gap-1 sm:justify-end">
									<Button
										type="button"
										variant="ghost"
										size="icon"
										aria-label={textGet("agenda.schedule.add_range")}
										title={textGet("agenda.schedule.add_range")}
										onClick={() => addRange(day)}
									>
										<Plus className="h-4 w-4" />
									</Button>
									<CopyDayButton
										from={day}
										onCopy={(targets) => copyDay(day, targets)}
									/>
								</div>
							)}
						</div>
					);
				})}
			</div>
			{canEdit && (
				<div className="flex flex-wrap items-center justify-between gap-3">
					<p className="text-sm text-muted-foreground">
						{textGet("agenda.schedule.hint")}
					</p>
					<Button
						type="button"
						onClick={handleSave}
						disabled={saving || hasErrors || !dirty}
					>
						{saving ? (
							<Loader2 className="mr-2 h-4 w-4 animate-spin" />
						) : (
							<Save className="mr-2 h-4 w-4" />
						)}
						{textGet("agenda.schedule.save")}
					</Button>
				</div>
			)}
		</div>
	);
}
