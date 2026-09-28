/**
 * Calendar dates without a time of day (birthdays, "next appointment on",
 * task due dates). The API sends them as "2026-04-23" or, from a Postgres
 * `date` column, as "2026-04-23T00:00:00Z". Both mean the 23rd wherever the
 * user is, but `new Date(...)` reads them as UTC midnight, which is the 22nd
 * at 19:00 in Ecuador (UTC-5).
 */

const DATE_ONLY = /^(\d{4})-(\d{2})-(\d{2})$/;
const DATE_PREFIX = /^(\d{4})-(\d{2})-(\d{2})/;

function localDate(match: RegExpExecArray | null): Date | undefined {
	if (!match) return undefined;
	const [, y, m, d] = match;
	return new Date(Number(y), Number(m) - 1, Number(d));
}

/** A bare "YYYY-MM-DD" as local midnight of that day; anything else: undefined. */
export function parseBareDate(value: string): Date | undefined {
	return localDate(DATE_ONLY.exec(value));
}

/**
 * The calendar day of a date-only value, as local midnight of that day. Takes
 * the "YYYY-MM-DD" part of a string and ignores any time or offset after it,
 * so use it only for fields that are dates, never for timestamps.
 */
export function parseDateOnly(
	value: string | Date | null | undefined,
): Date | undefined {
	if (!value) return undefined;
	if (value instanceof Date) return value;
	return localDate(DATE_PREFIX.exec(value));
}

/** "YYYY-MM-DD" of the day `date` falls on in the local time zone. */
export function toDateOnlyString(date: Date): string {
	const y = String(date.getFullYear()).padStart(4, "0");
	const m = String(date.getMonth() + 1).padStart(2, "0");
	const d = String(date.getDate()).padStart(2, "0");
	return `${y}-${m}-${d}`;
}
