import { useText } from "@pengi/shared";
import { cn } from "@pengi/ui";
import type { ReactNode } from "react";

interface FormSectionProps {
	title: string;
	description: string;
	children: ReactNode;
}

/** A titled group of fields; consecutive sections are divided by a border. */
export function FormSection({
	title,
	description,
	children,
}: FormSectionProps) {
	return (
		<section className="space-y-4 border-t pt-5 first:border-t-0 first:pt-0">
			<div>
				<h3 className="text-sm font-medium">{title}</h3>
				<p className="text-xs text-muted-foreground">{description}</p>
			</div>
			{children}
		</section>
	);
}

export const TIERS = [1, 2, 3] as const;
export type Tier = (typeof TIERS)[number];

interface TierSelectProps {
	value: Tier;
	onChange: (tier: Tier) => void;
}

/** Segmented control to pick the plan tier (Base / Intermedio / Premium). */
export function TierSelect({ value, onChange }: TierSelectProps) {
	const { textGet } = useText();
	return (
		<fieldset className="space-y-2">
			<legend className="mb-2 text-sm font-medium leading-none">
				{textGet("backoffice.plans.col.tier")}
			</legend>
			<div className="block">
				<div className="inline-flex rounded-md border p-1">
					{TIERS.map((t) => (
						<button
							key={t}
							type="button"
							aria-pressed={value === t}
							onClick={() => onChange(t)}
							className={cn(
								"rounded-sm px-4 py-1.5 text-sm font-medium transition-colors",
								value === t
									? "bg-primary text-primary-foreground"
									: "text-muted-foreground hover:bg-muted",
							)}
						>
							{textGet(`backoffice.plans.tier.${t}`)}
						</button>
					))}
				</div>
			</div>
			<p className="text-xs text-muted-foreground">
				{textGet("backoffice.plans.tier.hint")}
			</p>
		</fieldset>
	);
}
