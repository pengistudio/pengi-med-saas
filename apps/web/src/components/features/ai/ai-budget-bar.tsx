import { useText } from "@pengi/shared";
import { Card, CardContent, cn, Text } from "@pengi/ui";
import React from "react";
import { type AIBudget, getAIStatus } from "@/api/ai-service";
import { AI_BUDGET_WARNING_PERCENT } from "@/lib/ai-budget";

/**
 * "Uso de IA este mes": how much of the plan's monthly AI budget the clinic
 * has used, as a percentage. Shared by summary, scribe and the WhatsApp bot.
 * Never shows dollars or tokens. Hidden while loading, with no key, or when
 * the plan has no cap.
 */
export function AIBudgetBar({
	variant = "plain",
}: {
	/** `card`: a compact dashboard card, shown only once the clinic turned AI on. */
	variant?: "plain" | "card";
}) {
	const { formatDate } = useText();
	const [budget, setBudget] = React.useState<AIBudget | null>(null);

	React.useEffect(() => {
		let cancelled = false;
		getAIStatus().then((res) => {
			if (cancelled || !res.success) return;
			if (variant === "card" && !res.data.enabled) return;
			setBudget(res.data.budget);
		});
		return () => {
			cancelled = true;
		};
	}, [variant]);

	if (!budget || budget.unlimited) return null;
	const warning = budget.percent >= AI_BUDGET_WARNING_PERCENT;
	const values = {
		percent: budget.percent,
		date: formatDate(budget.period_end, "day-month"),
	};

	const compact = variant === "card";
	const bar = (
		<div className="grid gap-2">
			<div className="flex items-baseline justify-between gap-2">
				<span className="text-sm font-medium">
					<Text uuid="ai.budget.title" />
				</span>
				<span
					className={cn(
						"font-semibold tabular-nums",
						compact ? "text-base" : "text-2xl",
					)}
				>
					{budget.percent}%
				</span>
			</div>
			<div
				className="h-2 overflow-hidden rounded-full bg-muted"
				role="progressbar"
				aria-valuemin={0}
				aria-valuemax={100}
				aria-valuenow={budget.percent}
			>
				<div
					className={cn(
						"h-full rounded-full transition-all",
						budget.reached
							? "bg-destructive"
							: warning
								? "bg-amber-500"
								: "bg-primary",
					)}
					style={{ width: `${budget.percent}%` }}
				/>
			</div>
			<p
				className={cn(
					"text-xs",
					budget.reached ? "text-destructive" : "text-muted-foreground",
				)}
			>
				{budget.reached ? (
					<Text uuid="ai.budget.reached" values={values} />
				) : (
					<Text uuid="ai.budget.hint" values={values} />
				)}
			</p>
		</div>
	);
	if (!compact) return bar;
	return (
		<Card className="py-0">
			<CardContent className="px-4 py-3">{bar}</CardContent>
		</Card>
	);
}
