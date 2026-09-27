import { useMessageStore, useText } from "@pengi/shared";
import { Popover, PopoverContent, PopoverTrigger } from "@pengi/ui";
import {
	AlertTriangle,
	CheckCircle2,
	ChevronDown,
	ChevronRight,
	Clock,
	FilePen,
	XCircle,
} from "lucide-react";
import type React from "react";
import { useNavigate } from "react-router";
import type {
	DashboardDraft,
	DashboardPatientRef,
	SubscriptionInfo,
} from "@/api/clinical-service";
import { formatRelativeTime } from "@/lib/notification-text";
import { cn } from "@/lib/utils";

/** Days before expiry from which the plan shows up here. */
export const SUBSCRIPTION_WARNING_DAYS = 30;

type Tone = "danger" | "warning";

const TONE_STYLES: Record<Tone, { chip: string; icon: string }> = {
	danger: {
		chip: "border-destructive/30 bg-destructive/5 hover:bg-destructive/10",
		icon: "text-destructive",
	},
	warning: {
		chip: "border-amber-500/30 bg-amber-500/5 hover:bg-amber-500/10",
		icon: "text-amber-600",
	},
};

const chipClass = (tone: Tone) =>
	cn(
		"inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition-colors",
		"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
		TONE_STYLES[tone].chip,
	);

/** "{count}" filled into the singular or plural form of a key. */
function useCountLabel() {
	const { textGet } = useText();
	return (key: string, count: number) =>
		textGet(`${key}.${count === 1 ? "one" : "other"}`).replace(
			"{count}",
			String(count),
		);
}

function ChipContent({
	icon: Icon,
	tone,
	label,
	trailing,
}: {
	icon: React.ComponentType<{ className?: string }>;
	tone: Tone;
	label: string;
	trailing: React.ReactNode;
}) {
	return (
		<>
			<Icon className={cn("h-4 w-4 shrink-0", TONE_STYLES[tone].icon)} />
			<span>{label}</span>
			{trailing}
		</>
	);
}

/** A chip that opens a short list of the items it counts. */
function ListChip({
	icon,
	tone,
	label,
	children,
}: {
	icon: React.ComponentType<{ className?: string }>;
	tone: Tone;
	label: string;
	children: React.ReactNode;
}) {
	return (
		<Popover>
			<PopoverTrigger className={chipClass(tone)}>
				<ChipContent
					icon={icon}
					tone={tone}
					label={label}
					trailing={
						<ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
					}
				/>
			</PopoverTrigger>
			<PopoverContent align="start" className="w-72 p-1">
				{children}
			</PopoverContent>
		</Popover>
	);
}

function ListLink({
	primary,
	secondary,
	onClick,
}: {
	primary: string;
	secondary?: string;
	onClick: () => void;
}) {
	return (
		<button
			type="button"
			onClick={onClick}
			className="flex w-full items-center justify-between gap-3 rounded-md px-3 py-2 text-left text-sm hover:bg-muted focus-visible:bg-muted focus-visible:outline-none"
		>
			<span className="truncate font-medium">{primary}</span>
			{secondary && (
				<span className="shrink-0 text-xs text-muted-foreground">
					{secondary}
				</span>
			)}
		</button>
	);
}

interface AttentionStripProps {
	criticalCount: number;
	criticalPatients: DashboardPatientRef[];
	draftsCount: number;
	drafts: DashboardDraft[];
	/** undefined when billing is off for the tenant. */
	failedInvoices?: number;
	subscription?: SubscriptionInfo;
}

/**
 * What needs someone to act today, each item one click from where it gets
 * fixed. When nothing does, it says so in one line.
 */
export function AttentionStrip({
	criticalCount,
	criticalPatients,
	draftsCount,
	drafts,
	failedInvoices,
	subscription,
}: AttentionStripProps) {
	const { textGet } = useText();
	const countLabel = useCountLabel();
	const navigate = useNavigate();
	const lang = useMessageStore((s) => s.lang);

	const planExpiring =
		subscription !== undefined &&
		subscription.days_left <= SUBSCRIPTION_WARNING_DAYS;
	const hasInvoices = failedInvoices !== undefined && failedInvoices > 0;
	const nothingPending =
		criticalCount === 0 && draftsCount === 0 && !hasInvoices && !planExpiring;

	if (nothingPending) {
		return (
			<p className="flex items-center gap-2 rounded-xl border bg-card px-4 py-3 text-sm text-muted-foreground">
				<CheckCircle2 className="h-4 w-4 text-emerald-600" />
				{textGet("dashboard.attention.none")}
			</p>
		);
	}

	return (
		<section
			aria-labelledby="dashboard-attention"
			className="flex flex-col gap-3 rounded-xl border bg-card px-4 py-3 sm:flex-row sm:items-center"
		>
			<h2
				id="dashboard-attention"
				className="shrink-0 text-sm font-semibold sm:w-40"
			>
				{textGet("dashboard.attention.title")}
			</h2>
			<div className="flex flex-wrap gap-2">
				{criticalCount > 0 && (
					<ListChip
						icon={AlertTriangle}
						tone="danger"
						label={countLabel("dashboard.attention.critical", criticalCount)}
					>
						{criticalPatients.map((p) => (
							<ListLink
								key={p.id}
								primary={p.name}
								onClick={() => navigate(`/clinical/medical-records/${p.id}`)}
							/>
						))}
						{criticalCount > criticalPatients.length && (
							<ListLink
								primary={textGet("dashboard.attention.critical.view_all")}
								onClick={() => navigate("/clinical")}
							/>
						)}
					</ListChip>
				)}

				{draftsCount > 0 && (
					<ListChip
						icon={FilePen}
						tone="warning"
						label={countLabel("dashboard.attention.drafts", draftsCount)}
					>
						{drafts.map((d) => (
							<ListLink
								key={d.patient_id}
								primary={d.patient_name}
								secondary={formatRelativeTime(d.updated_at, lang)}
								onClick={() =>
									navigate(`/clinical/medical-records/${d.patient_id}`)
								}
							/>
						))}
					</ListChip>
				)}

				{hasInvoices && (
					<button
						type="button"
						className={chipClass("danger")}
						onClick={() => navigate("/billing?status=failed,rejected")}
					>
						<ChipContent
							icon={XCircle}
							tone="danger"
							label={countLabel(
								"dashboard.attention.invoices",
								failedInvoices ?? 0,
							)}
							trailing={
								<ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
							}
						/>
					</button>
				)}

				{planExpiring && subscription && (
					<button
						type="button"
						className={chipClass(
							subscription.days_left <= 7 ? "danger" : "warning",
						)}
						onClick={() => navigate("/subscription")}
					>
						<ChipContent
							icon={Clock}
							tone={subscription.days_left <= 7 ? "danger" : "warning"}
							label={
								subscription.days_left === 0
									? textGet("dashboard.attention.plan.expired")
									: countLabel(
											"dashboard.attention.plan",
											subscription.days_left,
										)
							}
							trailing={
								<ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
							}
						/>
					</button>
				)}
			</div>
		</section>
	);
}
