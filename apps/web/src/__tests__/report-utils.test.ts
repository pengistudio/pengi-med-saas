import { describe, expect, it } from "vitest";
import {
	currentMonthRange,
	reportFilename,
	toISODate,
	validateRange,
} from "@/pages/reports/report-utils";

describe("report utils", () => {
	it("formats a local date as YYYY-MM-DD", () => {
		expect(toISODate(new Date(2026, 0, 5))).toBe("2026-01-05");
	});

	it("defaults to the current month through today", () => {
		expect(currentMonthRange(new Date(2026, 10, 11))).toEqual({
			from: "2026-11-01",
			to: "2026-11-11",
		});
	});

	it("validates the range", () => {
		expect(validateRange("2026-11-01", "2026-11-11")).toBeNull();
		expect(validateRange("2026-11-11", "2026-11-11")).toBeNull();
		expect(validateRange("2026-11-12", "2026-11-11")).toBe("order");
		expect(validateRange("", "2026-11-11")).toBe("invalid");
		// 366 days inclusive is the limit
		expect(validateRange("2026-01-01", "2027-01-01")).toBeNull();
		expect(validateRange("2026-01-01", "2027-01-02")).toBe("too_long");
	});

	it("names the downloaded file", () => {
		expect(
			reportFilename(
				"revenue",
				{ from: "2026-11-01", to: "2026-11-11" },
				"csv",
			),
		).toBe("reports-revenue-2026-11-01_2026-11-11.csv");
	});
});
