import type { AIBudget } from "@/api/ai-service";

/** From this share of the month's AI budget, the clinic is warned it is running low. */
export const AI_BUDGET_WARNING_PERCENT = 80;

/**
 * Which message to show about the AI budget (`ai.budget.reached` or
 * `ai.budget.low`; literal keys stay at the call sites), or null while there is plenty
 * left (or no cap). The clinic only ever sees a percentage, never dollars.
 */
export function aiBudgetNotice(
	budget: AIBudget | undefined,
): "reached" | "low" | null {
	if (!budget || budget.unlimited) return null;
	if (budget.reached) return "reached";
	return budget.percent >= AI_BUDGET_WARNING_PERCENT ? "low" : null;
}
