import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useText } from "@/hooks/use-text";

export const PLAN_LIMIT_KEYS = [
	"max_users",
	"max_patients",
	"max_offices",
] as const;
export type PlanLimitKey = (typeof PLAN_LIMIT_KEYS)[number];
export type PlanLimits = Partial<Record<PlanLimitKey, number | null>>;

const LIMIT_CONFIGS: {
	key: PlanLimitKey;
	labelKey: string;
	descKey: string;
}[] = [
	{
		key: "max_users",
		labelKey: "backoffice.plans.limits.max_users",
		descKey: "backoffice.plans.limits.max_users.desc",
	},
	{
		key: "max_patients",
		labelKey: "backoffice.plans.limits.max_patients",
		descKey: "backoffice.plans.limits.max_patients.desc",
	},
	{
		key: "max_offices",
		labelKey: "backoffice.plans.limits.max_offices",
		descKey: "backoffice.plans.limits.max_offices.desc",
	},
];

interface PlanLimitsEditorProps {
	limits: PlanLimits;
	onChange: (limits: PlanLimits) => void;
}

export function PlanLimitsEditor({ limits, onChange }: PlanLimitsEditorProps) {
	const { textGet } = useText();

	const isUnlimited = (key: PlanLimitKey) => {
		const v = limits[key];
		return v === undefined || v === null || v === -1;
	};

	const handleValueChange = (key: PlanLimitKey, raw: string) => {
		const num = raw === "" ? null : Math.max(1, Number.parseInt(raw, 10));
		onChange({ ...limits, [key]: num });
	};

	const handleUnlimitedToggle = (key: PlanLimitKey, checked: boolean) => {
		onChange({ ...limits, [key]: checked ? -1 : 1 });
	};

	return (
		<div className="space-y-3">
			<Label>{textGet("backoffice.plans.limits.title")}</Label>
			<div className="border rounded-md divide-y">
				{LIMIT_CONFIGS.map(({ key, labelKey, descKey }) => (
					<div key={key} className="flex items-center gap-4 px-4 py-3">
						<div className="flex-1 min-w-0">
							<p className="text-sm font-medium">{textGet(labelKey)}</p>
							<p className="text-xs text-muted-foreground">
								{textGet(descKey)}
							</p>
						</div>
						<div className="flex items-center gap-3 shrink-0">
							<Input
								type="number"
								min={1}
								className="w-24 text-center"
								disabled={isUnlimited(key)}
								value={isUnlimited(key) ? "" : (limits[key] ?? "")}
								onChange={(e) => handleValueChange(key, e.target.value)}
								placeholder="—"
							/>
							<div className="flex items-center gap-1.5">
								<Checkbox
									checked={isUnlimited(key)}
									onCheckedChange={(checked) =>
										handleUnlimitedToggle(key, !!checked)
									}
								/>
								<span className="text-xs text-muted-foreground whitespace-nowrap">
									{textGet("backoffice.plans.limits.unlimited")}
								</span>
							</div>
						</div>
					</div>
				))}
			</div>
		</div>
	);
}
