import type {
	AppointmentType,
	ScheduleBlock,
	TimeRange,
} from "@/api/agenda-service";
import {
	END_HOUR,
	minutesToTime,
	START_HOUR,
	timeToMinutes,
} from "@/components/features/appointments/appointment-utils";

/** Weekdays in the order the editor lists them: Monday first (0 = Sunday). */
export const WEEKDAY_ORDER = [1, 2, 3, 4, 5, 6, 0] as const;

export const WEEKDAY_KEYS: Record<number, string> = {
	0: "agenda.weekday.0",
	1: "agenda.weekday.1",
	2: "agenda.weekday.2",
	3: "agenda.weekday.3",
	4: "agenda.weekday.4",
	5: "agenda.weekday.5",
	6: "agenda.weekday.6",
};

const TIME = /^([01]\d|2[0-3]):([0-5]\d)$|^24:00$/;

/** "HH:MM" (or "24:00") on a 5-minute step. */
export function isStepTime(time: string) {
	return TIME.test(time) && timeToMinutes(time) % 5 === 0;
}

// A native time input can't show "24:00" (end of day): it shows "00:00", and
// an end of "00:00" means midnight at the end of the day.
export const endToInput = (time: string) => (time === "24:00" ? "00:00" : time);
export const endFromInput = (time: string) =>
	time === "00:00" ? "24:00" : time;

/** The week being edited: ranges per weekday (0..6). */
export type WeekDraft = Record<number, TimeRange[]>;

/**
 * The first problem of each weekday, as an i18n key: a bad or off-step time,
 * start not before end, or two overlapping ranges (touching is fine).
 */
export function validateWeek(week: WeekDraft): Record<number, string> {
	const errors: Record<number, string> = {};
	for (const [day, ranges] of Object.entries(week)) {
		const bad = ranges.some(
			(r) =>
				!isStepTime(r.start_time) ||
				!isStepTime(r.end_time) ||
				timeToMinutes(r.start_time) >= timeToMinutes(r.end_time),
		);
		if (bad) {
			errors[Number(day)] = "agenda.schedule.error.invalid_time";
			continue;
		}
		const sorted = [...ranges].sort(
			(a, b) => timeToMinutes(a.start_time) - timeToMinutes(b.start_time),
		);
		for (let i = 1; i < sorted.length; i++) {
			if (
				timeToMinutes(sorted[i].start_time) <
				timeToMinutes(sorted[i - 1].end_time)
			) {
				errors[Number(day)] = "agenda.schedule.error.overlap";
				break;
			}
		}
	}
	return errors;
}

/** `time` plus `minutes`, kept inside the day (5-minute step). */
export function addMinutes(time: string, minutes: number) {
	return minutesToTime(Math.min(timeToMinutes(time) + minutes, 23 * 60 + 55));
}

/**
 * Types offered for a doctor: those open to every doctor or listing this one
 * (with no doctor, only the unrestricted ones). Inactive types are left out
 * except `keepId`, the appointment's current type.
 */
export function typesForDoctor(
	types: AppointmentType[],
	doctorId: number | null | undefined,
	keepId?: number | null,
) {
	return types.filter(
		(t) =>
			(t.active || t.ID === keepId) &&
			((t.doctor_ids ?? []).length === 0 ||
				(!!doctorId && t.doctor_ids.includes(doctorId))),
	);
}

/** A shaded stretch of a column, in minutes from midnight. */
export interface ShadeRange {
	start: number;
	end: number;
	kind: "outside" | "blocked";
	reason?: string;
}

/**
 * What to shade in one agenda column: the time outside the working ranges
 * (only when the doctor has a schedule) and the blocked time, clipped to the
 * visible hours. Blocks are drawn over the "outside" stretches.
 */
export function computeShading({
	ranges,
	blocks,
	hasSchedule,
}: {
	ranges: TimeRange[];
	blocks: ScheduleBlock[];
	hasSchedule: boolean;
}): ShadeRange[] {
	const dayStart = START_HOUR * 60;
	const dayEnd = END_HOUR * 60;
	const clip = (start: number, end: number) => ({
		start: Math.max(start, dayStart),
		end: Math.min(end, dayEnd),
	});
	const result: ShadeRange[] = [];

	if (hasSchedule) {
		const merged = ranges
			.map((r) => ({
				start: timeToMinutes(r.start_time),
				end: timeToMinutes(r.end_time),
			}))
			.sort((a, b) => a.start - b.start)
			.reduce<{ start: number; end: number }[]>((acc, r) => {
				const last = acc[acc.length - 1];
				if (last && r.start <= last.end) last.end = Math.max(last.end, r.end);
				else acc.push({ ...r });
				return acc;
			}, []);
		let cursor = dayStart;
		for (const r of merged) {
			if (r.start > cursor) {
				result.push({ ...clip(cursor, r.start), kind: "outside" });
			}
			cursor = Math.max(cursor, r.end);
		}
		if (cursor < dayEnd)
			result.push({ ...clip(cursor, dayEnd), kind: "outside" });
	}

	for (const block of blocks) {
		const full = !block.start_time || !block.end_time;
		const span = full
			? { start: dayStart, end: dayEnd }
			: clip(timeToMinutes(block.start_time), timeToMinutes(block.end_time));
		result.push({ ...span, kind: "blocked", reason: block.reason });
	}

	return result.filter((r) => r.end > r.start);
}
