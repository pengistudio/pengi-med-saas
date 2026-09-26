import { describe, expect, it } from "vitest";
import { safeNext } from "./next";

describe("safeNext", () => {
	it("keeps paths inside the app", () => {
		expect(safeNext("/plans/edit/3?tab=1")).toBe("/plans/edit/3?tab=1");
	});

	it("falls back to home for missing or external targets", () => {
		for (const next of [
			null,
			"",
			"https://evil.example",
			"//evil.example",
			"/\\evil.example",
			"plans",
		]) {
			expect(safeNext(next)).toBe("/");
		}
	});
});
