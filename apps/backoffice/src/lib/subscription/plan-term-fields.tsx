import { useText } from "@pengi/shared";
import {
	cn,
	Input,
	Label,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@pengi/ui";
import React from "react";
import type { Plan } from "@/api/plan-service";
import { sortedPricings, suggestExpiry, todayInEcuador } from "./term";

/** A plan and the date (YYYY-MM-DD, Ecuador) the subscription expires on. */
export type PlanTerm = { planCode: string; expiresAt: string };

/**
 * Plan, period and expiry of a subscription. Picking a plan or one of its
 * periods suggests the expiry (from today, in Ecuador); the date stays
 * editable.
 */
export function PlanTermFields({
	plans,
	value,
	onChange,
	suggestOnPlanChange = true,
	now,
}: {
	plans: Plan[];
	value: PlanTerm;
	onChange: (term: PlanTerm) => void;
	/** Editing an existing subscription keeps its expiry when the plan changes. */
	suggestOnPlanChange?: boolean;
	/** For tests: the moment "today" is taken from. */
	now?: Date;
}) {
	const { textGet, formatMoney } = useText();
	const expiresId = React.useId();
	const pricings = sortedPricings(plans.find((p) => p.code === value.planCode));
	const suggested = (months: number) => suggestExpiry(months, now);
	const selectedMonths = pricings.find(
		(p) => suggested(p.months) === value.expiresAt,
	)?.months;

	const choosePlan = (planCode: string) => {
		if (!suggestOnPlanChange) {
			onChange({ ...value, planCode });
			return;
		}
		const first = sortedPricings(plans.find((p) => p.code === planCode))[0];
		onChange({ planCode, expiresAt: suggested(first?.months ?? 1) });
	};

	return (
		<>
			<div className="space-y-2">
				<Label>{textGet("backoffice.subscriptions.col.plan")}</Label>
				<Select
					value={value.planCode}
					onValueChange={(v) => v && choosePlan(v)}
				>
					<SelectTrigger>
						<SelectValue
							placeholder={textGet("backoffice.subscriptions.select.plan")}
						/>
					</SelectTrigger>
					<SelectContent>
						{plans.map((p) => (
							<SelectItem key={p.code} value={p.code}>
								{p.name}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
			</div>

			{pricings.length > 0 && (
				<div className="space-y-2">
					<Label>{textGet("backoffice.plans.pricings.title")}</Label>
					<div className="flex flex-wrap gap-2">
						{pricings.map((p) => (
							<button
								key={p.months}
								type="button"
								onClick={() =>
									onChange({ ...value, expiresAt: suggested(p.months) })
								}
								className={cn(
									"px-3 py-1.5 rounded-md text-sm font-medium transition-colors border",
									selectedMonths === p.months
										? "bg-primary text-primary-foreground border-primary"
										: "bg-muted text-muted-foreground border-transparent hover:bg-muted/80",
								)}
							>
								{textGet(`subscription.plans.period.${p.months}`)} —{" "}
								{formatMoney(p.price)}
							</button>
						))}
					</div>
				</div>
			)}

			<div className="space-y-2">
				<Label htmlFor={expiresId}>
					{textGet("backoffice.subscriptions.col.expires")}
				</Label>
				<Input
					id={expiresId}
					type="date"
					value={value.expiresAt}
					min={todayInEcuador(now)}
					onChange={(e) => onChange({ ...value, expiresAt: e.target.value })}
					required
				/>
			</div>
		</>
	);
}
