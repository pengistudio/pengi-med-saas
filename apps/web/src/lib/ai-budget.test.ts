import { describe, expect, it } from "vitest";
import type { AIBudget } from "@/api/ai-service";
import { aiBudgetNotice } from "./ai-budget";

const budget = (over: Partial<AIBudget>): AIBudget => ({
	percent: 0,
	unlimited: false,
	reached: false,
	period_start: "",
	period_end: "",
	...over,
});

describe("aiBudgetNotice", () => {
	it("stays quiet while there is plenty left", () => {
		expect(aiBudgetNotice(budget({ percent: 79 }))).toBeNull();
		expect(aiBudgetNotice(undefined)).toBeNull();
	});

	it("warns from 80% and reports when it ran out", () => {
		expect(aiBudgetNotice(budget({ percent: 80 }))).toBe("low");
		expect(aiBudgetNotice(budget({ percent: 100, reached: true }))).toBe(
			"reached",
		);
	});

	it("says nothing without a cap", () => {
		expect(aiBudgetNotice(budget({ unlimited: true, percent: 0 }))).toBeNull();
	});
});
