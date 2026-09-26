import { describe, expect, it } from "vitest";
import {
	addMonths,
	expiryDate,
	formatExpiry,
	sortedPricings,
	suggestExpiry,
	todayInEcuador,
} from "./term";

describe("subscription term", () => {
	it("today is the Ecuador calendar day, also late in the evening", () => {
		// 26 Sep 20:30 in Ecuador is already 27 Sep in UTC.
		expect(todayInEcuador(new Date("2026-09-27T01:30:00Z"))).toBe("2026-09-26");
	});

	it("adds months clamping to the last day of the month", () => {
		expect(addMonths("2026-01-31", 1)).toBe("2026-02-28");
		expect(addMonths("2028-01-31", 1)).toBe("2028-02-29");
		expect(addMonths("2026-08-31", 6)).toBe("2027-02-28");
		expect(addMonths("2026-11-30", 3)).toBe("2027-02-28");
		expect(addMonths("2026-01-15", 12)).toBe("2027-01-15");
	});

	it("suggests an expiry counted from today in Ecuador", () => {
		expect(suggestExpiry(1, new Date("2027-01-31T20:00:00-05:00"))).toBe(
			"2027-02-28",
		);
	});

	it("reads an expiry timestamp as its Ecuador date", () => {
		// End of 16 Nov in Ecuador, as the API now stores it.
		expect(expiryDate("2026-11-17T04:59:59Z")).toBe("2026-11-16");
		expect(formatExpiry("2026-11-17T04:59:59Z", "es-EC")).toBe("16/11/2026");
	});

	it("sorts a plan's pricings by months", () => {
		expect(
			sortedPricings({
				pricings: [
					{ months: 12, price: 300 },
					{ months: 1, price: 30 },
					{ months: 3, price: 90 },
				],
			}).map((p) => p.months),
		).toEqual([1, 3, 12]);
		expect(sortedPricings(undefined)).toEqual([]);
	});
});
