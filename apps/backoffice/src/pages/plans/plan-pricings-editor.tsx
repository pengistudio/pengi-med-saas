import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useText } from "@/hooks/use-text";

export const PERIOD_MONTHS = [1, 3, 6, 9, 12] as const;
export type PeriodMonths = (typeof PERIOD_MONTHS)[number];

export interface PricingEntry {
	months: PeriodMonths;
	price: number;
}

// Internal state: months → price (null = disabled)
export type PricingsState = Partial<Record<PeriodMonths, number | null>>;

interface PlanPricingsEditorProps {
	pricings: PricingsState;
	onChange: (pricings: PricingsState) => void;
}

export function pricingsStateToArray(state: PricingsState): PricingEntry[] {
	return PERIOD_MONTHS.filter(
		(m) => state[m] !== null && state[m] !== undefined && (state[m] ?? 0) > 0,
	).map((m) => ({ months: m, price: state[m] as number }));
}

export function arrayToPricingsState(
	entries: { months: number; price: number }[],
): PricingsState {
	const state: PricingsState = {};
	for (const e of entries) {
		if (PERIOD_MONTHS.includes(e.months as PeriodMonths)) {
			state[e.months as PeriodMonths] = e.price;
		}
	}
	return state;
}

export function PlanPricingsEditor({
	pricings,
	onChange,
}: PlanPricingsEditorProps) {
	const { textGet } = useText();

	const isEnabled = (months: PeriodMonths) =>
		pricings[months] !== null && pricings[months] !== undefined;

	const handleToggle = (months: PeriodMonths, checked: boolean) => {
		onChange({ ...pricings, [months]: checked ? 0 : null });
	};

	const handlePrice = (months: PeriodMonths, raw: string) => {
		const val = raw === "" ? 0 : Math.max(0, Number.parseFloat(raw));
		onChange({ ...pricings, [months]: val });
	};

	return (
		<div className="space-y-3 border-t pt-4">
			<div>
				<Label>{textGet("backoffice.plans.pricings.title")}</Label>
				<p className="text-xs text-muted-foreground mt-0.5">
					{textGet("backoffice.plans.pricings.description")}
				</p>
			</div>
			<div className="border rounded-md divide-y">
				{PERIOD_MONTHS.map((months) => {
					const enabled = isEnabled(months);
					const perMonth = enabled
						? ((pricings[months] ?? 0) / months).toFixed(2)
						: null;
					return (
						<div key={months} className="flex items-center gap-4 px-4 py-3">
							<div className="flex items-center gap-2 w-28 shrink-0">
								<Checkbox
									checked={enabled}
									onCheckedChange={(checked) => handleToggle(months, !!checked)}
								/>
								<span className="text-sm font-medium">
									{textGet(`subscription.plans.period.${months}`)}
								</span>
							</div>
							<div className="flex-1 flex items-center gap-2">
								<div className="relative w-32">
									<span className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground text-sm">
										$
									</span>
									<Input
										type="number"
										min={0}
										step={0.01}
										disabled={!enabled}
										value={enabled ? pricings[months] || "" : ""}
										onChange={(e) => handlePrice(months, e.target.value)}
										className="pl-6 w-32"
										placeholder="0.00"
									/>
								</div>
								{enabled && perMonth && months > 1 && (
									<span className="text-xs text-muted-foreground whitespace-nowrap">
										≈ ${perMonth} / mes
									</span>
								)}
							</div>
						</div>
					);
				})}
			</div>
		</div>
	);
}
