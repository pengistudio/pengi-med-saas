import { renderHook } from "@testing-library/react";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { parseDateOnly, toDateOnlyString } from "./date-only";
import { useMessageStore } from "./message-store";
import { useText } from "./use-text";

// Ecuador: UTC midnight is the previous day at 19:00.
const ORIGINAL_TZ = process.env.TZ;
beforeAll(() => {
	process.env.TZ = "America/Guayaquil";
});
afterAll(() => {
	process.env.TZ = ORIGINAL_TZ;
});

const day = (date: Date | undefined) =>
	date && [date.getFullYear(), date.getMonth() + 1, date.getDate()];

describe("date-only values in UTC-5", () => {
	it("runs in a time zone behind UTC", () => {
		// What the bug looked like: the 23rd read as the 22nd.
		expect(new Date("2026-04-23T00:00:00Z").getDate()).toBe(22);
	});

	it("parseDateOnly keeps the calendar day of a Postgres date column", () => {
		expect(day(parseDateOnly("2026-04-23T00:00:00Z"))).toEqual([2026, 4, 23]);
		expect(day(parseDateOnly("2026-04-23"))).toEqual([2026, 4, 23]);
	});

	it("parseDateOnly passes Dates through and rejects empty values", () => {
		const date = new Date(2026, 3, 23);
		expect(parseDateOnly(date)).toBe(date);
		expect(parseDateOnly(null)).toBeUndefined();
		expect(parseDateOnly("")).toBeUndefined();
		expect(parseDateOnly("not a date")).toBeUndefined();
	});

	it("toDateOnlyString sends the day picked, not the UTC day", () => {
		expect(toDateOnlyString(new Date(2026, 3, 23))).toBe("2026-04-23");
		// 21:00 local is already the next day in UTC.
		expect(toDateOnlyString(new Date(2026, 3, 23, 21))).toBe("2026-04-23");
	});

	it("round-trips a date column value", () => {
		expect(
			toDateOnlyString(parseDateOnly("2026-04-23T00:00:00Z") as Date),
		).toBe("2026-04-23");
	});

	it("formatDate reads a bare YYYY-MM-DD as that day", () => {
		useMessageStore.setState({ lang: "en", messages: {} });
		const { result } = renderHook(() => useText());
		expect(result.current.formatDate("2026-04-10", "long")).toBe(
			"April 10, 2026",
		);
		// Timestamps are still instants, shown in local time.
		expect(result.current.formatDate("2026-04-10T00:00:00Z", "long")).toBe(
			"April 9, 2026",
		);
	});
});
