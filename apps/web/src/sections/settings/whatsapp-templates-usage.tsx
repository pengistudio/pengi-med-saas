import { useText } from "@pengi/shared";
import { Button, cn, Skeleton, Text } from "@pengi/ui";
import { RefreshCw } from "lucide-react";
import React from "react";
import {
	getWhatsAppTemplates,
	getWhatsAppUsage,
	type WhatsAppTemplate,
	type WhatsAppTemplateStatus,
	type WhatsAppUsage,
} from "@/api/whatsapp-service";
import { usageSummary } from "@/lib/whatsapp-templates";

export const TEMPLATE_STYLES: Record<WhatsAppTemplateStatus, string> = {
	APPROVED: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
	PENDING: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
	REJECTED: "bg-destructive/10 text-destructive",
	PAUSED: "bg-destructive/10 text-destructive",
	DISABLED: "bg-destructive/10 text-destructive",
	"": "bg-muted text-muted-foreground",
};

export function StatusPill({ status }: { status: WhatsAppTemplateStatus }) {
	return (
		<span
			className={cn(
				"inline-flex w-fit items-center rounded-full px-2 py-0.5 text-xs font-medium",
				TEMPLATE_STYLES[status] ?? TEMPLATE_STYLES[""],
			)}
		>
			<Text uuid={`settings.whatsapp.template.status.${status || "none"}`} />
		</span>
	);
}

/**
 * Every template of the catalog with its status at Meta and the rejection
 * reason. `reloadKey` changes after a sync to re-read them.
 */
export function WhatsAppTemplateList({
	reloadKey,
	syncing,
	onSync,
}: {
	reloadKey: number;
	syncing: boolean;
	onSync: () => void;
}) {
	const [templates, setTemplates] = React.useState<WhatsAppTemplate[] | null>(
		null,
	);

	React.useEffect(() => {
		let cancelled = false;
		getWhatsAppTemplates().then((res) => {
			if (!cancelled && res.success) setTemplates(res.data ?? []);
		});
		return () => {
			cancelled = true;
		};
	}, [reloadKey]);

	return (
		<div className="grid gap-2 sm:col-span-2">
			<div className="flex flex-wrap items-center justify-between gap-2">
				<span className="text-xs text-muted-foreground">
					<Text uuid="settings.whatsapp.templates" />
				</span>
				<Button variant="ghost" size="sm" disabled={syncing} onClick={onSync}>
					<RefreshCw
						className={cn("mr-2 h-4 w-4", syncing && "animate-spin")}
					/>
					<Text uuid="settings.whatsapp.template.sync" />
				</Button>
			</div>
			{templates === null ? (
				<Skeleton className="h-24" />
			) : (
				<ul className="divide-y rounded-lg border">
					{templates.map((t) => (
						<li key={t.name} className="grid gap-1 px-3 py-2">
							<div className="flex flex-wrap items-center justify-between gap-2">
								<span className="grid">
									<span className="text-sm font-medium">
										<Text uuid={`whatsapp.template.name.${t.name}`} />
									</span>
									<span className="font-mono text-[11px] text-muted-foreground">
										{t.name}
									</span>
								</span>
								<span className="flex items-center gap-2">
									<span className="text-xs text-muted-foreground">
										<Text
											uuid={
												t.inbox
													? "settings.whatsapp.templates.inbox"
													: "settings.whatsapp.templates.reminder"
											}
										/>
									</span>
									<StatusPill status={t.status} />
								</span>
							</div>
							{t.reason && (
								<p className="text-xs text-destructive">{t.reason}</p>
							)}
						</li>
					))}
				</ul>
			)}
			{templates?.some((t) => t.status !== "APPROVED") && (
				<p className="text-xs text-muted-foreground">
					<Text uuid="settings.whatsapp.templates.hint" />
				</p>
			)}
		</div>
	);
}

/** "Mensajes este mes": templates sent against the plan's monthly cap. */
export function WhatsAppUsageBar({ reloadKey }: { reloadKey: number }) {
	const { formatDate } = useText();
	const [usage, setUsage] = React.useState<WhatsAppUsage | null>(null);

	React.useEffect(() => {
		let cancelled = false;
		getWhatsAppUsage().then((res) => {
			if (!cancelled && res.success) setUsage(res.data);
		});
		return () => {
			cancelled = true;
		};
	}, [reloadKey]);

	if (!usage) return null;
	const summary = usageSummary(usage);
	// period_end is exclusive: the first day of next month, when it resets.
	const values = {
		used: usage.used,
		limit: usage.limit,
		date: formatDate(usage.period_end, "day-month"),
	};

	return (
		<div className="grid gap-1.5 sm:col-span-2">
			<div className="flex items-baseline justify-between gap-2">
				<span className="text-xs text-muted-foreground">
					<Text uuid="settings.whatsapp.usage" />
				</span>
				<span className="text-sm font-medium tabular-nums">
					<Text
						uuid={
							summary.unlimited
								? "settings.whatsapp.usage.unlimited"
								: "settings.whatsapp.usage.value"
						}
						values={values}
					/>
				</span>
			</div>
			{!summary.unlimited && (
				<div
					className="h-2 overflow-hidden rounded-full bg-muted"
					role="progressbar"
					aria-valuemin={0}
					aria-valuemax={usage.limit}
					aria-valuenow={Math.min(usage.used, usage.limit)}
				>
					<div
						className={cn(
							"h-full rounded-full transition-all",
							summary.reached
								? "bg-destructive"
								: summary.warning
									? "bg-amber-500"
									: "bg-primary",
						)}
						style={{ width: `${summary.percent}%` }}
					/>
				</div>
			)}
			<p
				className={cn(
					"text-xs",
					summary.reached ? "text-destructive" : "text-muted-foreground",
				)}
			>
				<Text
					uuid={
						summary.reached
							? "settings.whatsapp.usage.reached"
							: "settings.whatsapp.usage.hint"
					}
					values={values}
				/>
			</p>
		</div>
	);
}
