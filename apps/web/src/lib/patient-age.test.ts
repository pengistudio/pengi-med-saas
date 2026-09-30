import { describe, expect, it } from "vitest";
import { ageInYears, birthDateFromAge } from "./patient-age";

const today = new Date(2026, 8, 29); // 29 Sep 2026

describe("ageInYears", () => {
	it("adds the year on the birthday itself", () => {
		expect(ageInYears(new Date(1968, 8, 29), today)).toBe(58);
		expect(ageInYears(new Date(1968, 8, 30), today)).toBe(57);
	});

	it("is null without a birth date", () => {
		expect(ageInYears(undefined, today)).toBeNull();
		expect(ageInYears("", today)).toBeNull();
		expect(ageInYears("0001-01-01T00:00:00Z", today)).toBeNull();
		expect(ageInYears("not a date", today)).toBeNull();
	});
});

describe("birthDateFromAge", () => {
	it("gives back the same age today", () => {
		for (const age of [0, 1, 34, 57, 90]) {
			expect(ageInYears(birthDateFromAge(age, today), today)).toBe(age);
		}
	});

	it("lands in the middle of the possible birth year", () => {
		expect(birthDateFromAge(57, today)).toEqual(new Date(1969, 2, 29));
	});
});
