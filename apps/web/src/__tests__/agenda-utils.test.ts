import { describe, expect, it } from "vitest";
import type { AppointmentType, ScheduleBlock } from "@/api/agenda-service";
import {
	addMinutes,
	computeShading,
	endFromInput,
	typesForDoctor,
	validateWeek,
} from "@/components/features/agenda/agenda-utils";

const block = (over: Partial<ScheduleBlock>): ScheduleBlock =>
	({
		ID: 1,
		tenant_id: 1,
		doctor_id: 3,
		start_date: "2026-10-19",
		end_date: "2026-10-19",
		start_time: "",
		end_time: "",
		reason: "",
		...over,
	}) as ScheduleBlock;

const type = (over: Partial<AppointmentType>): AppointmentType =>
	({
		ID: 1,
		tenant_id: 1,
		name: "Control",
		duration_minutes: 20,
		color: "",
		active: true,
		doctor_ids: [],
		...over,
	}) as AppointmentType;

describe("validateWeek", () => {
	it("accepts touching ranges and an end of 24:00", () => {
		expect(
			validateWeek({
				1: [
					{ start_time: "08:00", end_time: "12:00" },
					{ start_time: "12:00", end_time: "24:00" },
				],
			}),
		).toEqual({});
	});

	it("flags overlaps, off-step times and inverted ranges per day", () => {
		const errors = validateWeek({
			1: [
				{ start_time: "08:00", end_time: "12:00" },
				{ start_time: "11:00", end_time: "13:00" },
			],
			2: [{ start_time: "08:03", end_time: "09:00" }],
			3: [{ start_time: "10:00", end_time: "09:00" }],
			4: [],
		});
		expect(errors).toEqual({
			1: "agenda.schedule.error.overlap",
			2: "agenda.schedule.error.invalid_time",
			3: "agenda.schedule.error.invalid_time",
		});
	});
});

describe("computeShading", () => {
	it("shades outside the merged working ranges within the visible hours", () => {
		const shades = computeShading({
			ranges: [
				{ start_time: "08:00", end_time: "12:00" },
				{ start_time: "12:00", end_time: "14:00" },
				{ start_time: "16:00", end_time: "18:00" },
			],
			blocks: [],
			hasSchedule: true,
		});
		expect(shades).toEqual([
			{ start: 6 * 60, end: 8 * 60, kind: "outside" },
			{ start: 14 * 60, end: 16 * 60, kind: "outside" },
			{ start: 18 * 60, end: 24 * 60, kind: "outside" },
		]);
	});

	it("shades nothing outside for a doctor without schedule, but still blocks", () => {
		const shades = computeShading({
			ranges: [],
			blocks: [
				block({ start_time: "13:00", end_time: "15:00", reason: "Trámite" }),
			],
			hasSchedule: false,
		});
		expect(shades).toEqual([
			{ start: 13 * 60, end: 15 * 60, kind: "blocked", reason: "Trámite" },
		]);
	});

	it("a full-day block covers the whole column", () => {
		const shades = computeShading({
			ranges: [],
			blocks: [block({ reason: "Feriado" })],
			hasSchedule: false,
		});
		expect(shades).toEqual([
			{ start: 6 * 60, end: 24 * 60, kind: "blocked", reason: "Feriado" },
		]);
	});

	it("a working day with no ranges is all outside", () => {
		expect(
			computeShading({ ranges: [], blocks: [], hasSchedule: true }),
		).toEqual([{ start: 6 * 60, end: 24 * 60, kind: "outside" }]);
	});
});

describe("typesForDoctor", () => {
	const types = [
		type({ ID: 1, doctor_ids: [] }),
		type({ ID: 2, doctor_ids: [3] }),
		type({ ID: 3, doctor_ids: [4] }),
		type({ ID: 4, active: false }),
	];

	it("offers unrestricted and the doctor's own active types", () => {
		expect(typesForDoctor(types, 3).map((t) => t.ID)).toEqual([1, 2]);
	});

	it("without doctor, only unrestricted types", () => {
		expect(typesForDoctor(types, null).map((t) => t.ID)).toEqual([1]);
	});

	it("keeps the current inactive type but not one the doctor doesn't offer", () => {
		expect(typesForDoctor(types, 3, 4).map((t) => t.ID)).toEqual([1, 2, 4]);
		expect(typesForDoctor(types, 3, 3).map((t) => t.ID)).toEqual([1, 2]);
	});
});

describe("time helpers", () => {
	it("adds a duration without leaving the day", () => {
		expect(addMinutes("09:00", 40)).toBe("09:40");
		expect(addMinutes("23:30", 60)).toBe("23:55");
	});

	it("reads an end of 00:00 as midnight", () => {
		expect(endFromInput("00:00")).toBe("24:00");
		expect(endFromInput("18:00")).toBe("18:00");
	});
});
