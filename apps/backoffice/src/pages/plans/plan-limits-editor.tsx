import { useText } from "@pengi/shared";
import { Checkbox, Input } from "@pengi/ui";
import React from "react";

export const PLAN_LIMIT_KEYS = [
	"max_users",
	"max_patients",
	"max_offices",
	"max_whatsapp_messages",
	"max_ai_budget_usd",
] as const;

/**
 * Monthly AI budget in USD the API applies when the plan doesn't set it
 * (absent is NOT unlimited for AI: every use has a provider cost). Shared by
 * summary, scribe and bot; clinics only see the percentage used.
 */
export const DEFAULT_AI_LIMITS = {
	max_ai_budget_usd: 10,
} as const;
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
	{
		key: "max_whatsapp_messages",
		labelKey: "backoffice.plans.limits.max_whatsapp_messages",
		descKey: "backoffice.plans.limits.max_whatsapp_messages.desc",
	},
	{
		key: "max_ai_budget_usd",
		labelKey: "backoffice.plans.limits.max_ai_budget_usd",
		descKey: "backoffice.plans.limits.max_ai_budget_usd.desc",
	},
];

interface PlanLimitsEditorProps {
	limits: PlanLimits;
	onChange: (limits: PlanLimits) => void;
	/** Storage quota in MB (0 = no attachments). Rendered as the last row when given. */
	storageQuotaMb?: number;
	onStorageQuotaChange?: (mb: number) => void;
}

const formatGb = (mb: number) => String(Math.round((mb / 1024) * 100) / 100);

export function PlanLimitsEditor({
	limits,
	onChange,
	storageQuotaMb,
	onStorageQuotaChange,
}: PlanLimitsEditorProps) {
	const { textGet } = useText();
	const [gbText, setGbText] = React.useState(formatGb(storageQuotaMb ?? 0));

	// Sync from the prop when it changes externally (e.g. the plan finishes loading).
	React.useEffect(() => {
		const parsed = Number.parseFloat(gbText);
		const current = Number.isFinite(parsed) ? Math.round(parsed * 1024) : 0;
		if (current !== (storageQuotaMb ?? 0))
			setGbText(formatGb(storageQuotaMb ?? 0));
	}, [storageQuotaMb, gbText]);

	const isUnlimited = (key: PlanLimitKey) => {
		const v = limits[key];
		return v === undefined || v === null || v === -1;
	};

	const handleValueChange = (key: PlanLimitKey, raw: string) => {
		// The AI budget is in dollars (cents allowed); the rest are whole counts.
		const parsed =
			key === "max_ai_budget_usd"
				? Number.parseFloat(raw)
				: Number.parseInt(raw, 10);
		const num =
			raw === "" || Number.isNaN(parsed) ? null : Math.max(0.01, parsed);
		onChange({ ...limits, [key]: num });
	};

	const handleUnlimitedToggle = (key: PlanLimitKey, checked: boolean) => {
		onChange({ ...limits, [key]: checked ? -1 : 1 });
	};

	return (
		<div className="border rounded-md divide-y">
			{LIMIT_CONFIGS.map(({ key, labelKey, descKey }) => (
				<div key={key} className="flex items-center gap-4 px-4 py-3">
					<div className="flex-1 min-w-0">
						<p className="text-sm font-medium">{textGet(labelKey)}</p>
						<p className="text-xs text-muted-foreground">{textGet(descKey)}</p>
					</div>
					<div className="flex items-center gap-3 shrink-0">
						<Input
							type="number"
							min={key === "max_ai_budget_usd" ? 0.01 : 1}
							step={key === "max_ai_budget_usd" ? "any" : 1}
							className="w-24 text-center"
							disabled={isUnlimited(key)}
							value={isUnlimited(key) ? "" : (limits[key] ?? "")}
							onChange={(e) => handleValueChange(key, e.target.value)}
							placeholder="—"
						/>
						<div className="flex w-24 items-center gap-1.5">
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
			{storageQuotaMb !== undefined && onStorageQuotaChange && (
				<div className="flex items-center gap-4 px-4 py-3">
					<div className="flex-1 min-w-0">
						<p className="text-sm font-medium">
							{textGet("backoffice.plans.storage_quota")}
						</p>
						<p className="text-xs text-muted-foreground">
							{textGet("backoffice.plans.storage_quota.desc")}
						</p>
					</div>
					<div className="flex items-center gap-3 shrink-0">
						<div className="relative w-24">
							<Input
								type="number"
								min={0}
								step="any"
								className="w-24 pr-9 text-center"
								value={gbText}
								onChange={(e) => {
									const raw = e.target.value;
									setGbText(raw);
									const gb = Number.parseFloat(raw);
									if (raw === "") onStorageQuotaChange(0);
									else if (Number.isFinite(gb) && gb >= 0)
										onStorageQuotaChange(Math.round(gb * 1024));
								}}
								onBlur={() => setGbText(formatGb(storageQuotaMb))}
							/>
							<span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">
								GB
							</span>
						</div>
						<span className="w-24 text-xs text-muted-foreground">
							{textGet("backoffice.plans.storage_quota.none")}
						</span>
					</div>
				</div>
			)}
		</div>
	);
}
