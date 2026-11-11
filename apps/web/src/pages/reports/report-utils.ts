import { MAX_RANGE_DAYS, type ReportFilters } from "@/api/reports-service";

const DAY_MS = 24 * 60 * 60 * 1000;

const pad = (n: number) => String(n).padStart(2, "0");

/** YYYY-MM-DD of a local date (the format the filters and the API use). */
export const toISODate = (d: Date): string =>
	`${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;

/** First day of the current month through today. */
export const currentMonthRange = (
	now: Date = new Date(),
): Pick<ReportFilters, "from" | "to"> => ({
	from: toISODate(new Date(now.getFullYear(), now.getMonth(), 1)),
	to: toISODate(now),
});

const parseISODate = (value: string): Date | null => {
	const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
	if (!m) return null;
	return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
};

export type RangeError = "invalid" | "order" | "too_long";

/** Why a range can't be queried, or null when it can. */
export const validateRange = (from: string, to: string): RangeError | null => {
	const a = parseISODate(from);
	const b = parseISODate(to);
	if (!a || !b) return "invalid";
	if (a > b) return "order";
	const days = Math.round((b.getTime() - a.getTime()) / DAY_MS) + 1;
	return days > MAX_RANGE_DAYS ? "too_long" : null;
};

/** Name of a downloaded report file. */
export const reportFilename = (
	kind: string,
	f: Pick<ReportFilters, "from" | "to">,
	format: "csv" | "pdf",
) => `reports-${kind}-${f.from}_${f.to}.${format}`;

export const saveBlob = (blob: Blob, filename: string) => {
	const url = window.URL.createObjectURL(blob);
	const link = document.createElement("a");
	link.href = url;
	link.download = filename;
	document.body.appendChild(link);
	link.click();
	document.body.removeChild(link);
	window.URL.revokeObjectURL(url);
};
