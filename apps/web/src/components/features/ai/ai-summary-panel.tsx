import { useText } from "@pengi/shared";
import { Badge, Button } from "@pengi/ui";
import { CircleAlert, Loader2, RefreshCw, Sparkles } from "lucide-react";
import React from "react";
import { Link } from "react-router";
import {
	type AIStatus,
	type AISummary,
	type AISummaryItem,
	type AISummarySource,
	generatePatientAISummary,
	getAIStatus,
	getPatientAISummary,
} from "@/api/ai-service";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useEnabledFeatures } from "@/hooks/use-enabled-features";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";

/**
 * Whether the AI summary may be offered: the plan includes AI, and the user
 * holds USE_AI_SUMMARY and READ_MEDICAL_RECORD. The clinic's opt-in comes
 * from the API (`AIStatus.enabled`).
 */
export function useCanUseAISummary(): boolean {
	const planHasAI = useEnabledFeatures().ai === true;
	const { checkPermission } = usePermission();
	return (
		planHasAI &&
		checkPermission([
			PERMISSIONS.AI.PERMISSION_USE_AI_SUMMARY,
			PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_MEDICAL_RECORD,
		])
	);
}

interface AISummaryPanelProps {
	patientId: number;
	/** Start collapsed to a single row (consultation form). */
	compact?: boolean;
}

/**
 * AI draft summary of the patient's history. Hidden unless the plan includes
 * AI, the user may use it and the clinic opted in. Labeled as an AI draft:
 * not part of the record, never printed or signed. Each statement links to
 * the consultations it comes from.
 */
export function AISummaryPanel({ patientId, compact }: AISummaryPanelProps) {
	const allowed = useCanUseAISummary();
	const { textGet } = useText();
	const [status, setStatus] = React.useState<AIStatus | null>(null);
	const [summary, setSummary] = React.useState<AISummary | null>(null);
	const [loading, setLoading] = React.useState(false);
	const [error, setError] = React.useState<string | null>(null);
	const [open, setOpen] = React.useState(!compact);

	React.useEffect(() => {
		if (!allowed) return;
		let cancelled = false;
		getAIStatus().then((res) => {
			if (cancelled || !res.success) return;
			setStatus(res.data);
			if (!res.data.enabled) return;
			// A cached summary costs nothing: show it right away if still valid.
			getPatientAISummary(patientId).then((cached) => {
				if (!cancelled && cached.success) setSummary(cached.data);
			});
		});
		return () => {
			cancelled = true;
		};
	}, [allowed, patientId]);

	if (!allowed || !status?.enabled) return null;
	// No key on the server: nothing to generate, so hide the panel (a cached
	// summary from before still shows).
	if (!status.configured && !summary) return null;

	const quota = status.quotas.find((q) => q.function === "summary");

	async function handleGenerate() {
		setLoading(true);
		setError(null);
		const res = await generatePatientAISummary(patientId);
		setLoading(false);
		if (res.success) {
			setSummary(res.data);
			setOpen(true);
			if (!res.data.cached && quota) {
				setStatus((s) =>
					s
						? {
								...s,
								quotas: s.quotas.map((q) =>
									q.function === "summary" ? { ...q, used: q.used + 1 } : q,
								),
							}
						: s,
				);
			}
		} else {
			setError(res.data?.error_message || textGet("ai.summary.error"));
		}
	}

	return (
		<section
			aria-label={textGet("ai.summary.title")}
			className="rounded-lg border border-violet-200 bg-violet-50/40 p-4 dark:border-violet-900 dark:bg-violet-950/20"
		>
			<div className="flex flex-wrap items-center justify-between gap-2">
				<div className="flex flex-wrap items-center gap-2">
					<Sparkles className="size-4 text-violet-600 dark:text-violet-400" />
					<h2 className="text-sm font-semibold">
						{textGet("ai.summary.title")}
					</h2>
					<Badge variant="outline" className="font-medium">
						{textGet("ai.summary.draft_badge")}
					</Badge>
				</div>
				<div className="flex items-center gap-2">
					{quota && quota.limit >= 0 && (
						<span className="text-xs tabular-nums text-muted-foreground">
							{textGet("ai.summary.quota", {
								used: quota.used,
								limit: quota.limit,
							})}
						</span>
					)}
					{summary && compact && (
						<Button
							type="button"
							size="sm"
							variant="ghost"
							onClick={() => setOpen((v) => !v)}
						>
							{open ? textGet("ai.summary.hide") : textGet("ai.summary.show")}
						</Button>
					)}
					<Button
						type="button"
						size="sm"
						variant={summary ? "outline" : "default"}
						disabled={loading}
						onClick={handleGenerate}
					>
						{loading ? (
							<Loader2 className="animate-spin" />
						) : summary ? (
							<RefreshCw />
						) : (
							<Sparkles />
						)}
						{loading
							? textGet("ai.summary.generating")
							: summary
								? textGet("ai.summary.refresh")
								: textGet("ai.summary.generate")}
					</Button>
				</div>
			</div>

			{error && (
				<Alert variant="destructive" className="mt-3">
					<CircleAlert />
					<AlertDescription>{error}</AlertDescription>
				</Alert>
			)}

			{summary && open && <SummaryBody summary={summary} />}

			<p className="mt-3 text-xs text-muted-foreground">
				{textGet("ai.summary.disclaimer")}
			</p>
		</section>
	);
}

function SummaryBody({ summary }: { summary: AISummary }) {
	const { textGet, formatDateTime } = useText();
	const sourcesById = React.useMemo(() => {
		const map = new Map<number, AISummarySource>();
		for (const s of summary.sources) map.set(s.medical_record_id, s);
		return map;
	}, [summary.sources]);
	const c = summary.content;

	return (
		<div className="mt-3 grid gap-3 text-sm">
			{c.overview && <p>{c.overview}</p>}
			<ItemList
				title={textGet("ai.summary.section.alerts")}
				items={c.alerts}
				sources={sourcesById}
				tone="alert"
			/>
			<ItemList
				title={textGet("ai.summary.section.active_problems")}
				items={c.active_problems}
				sources={sourcesById}
			/>
			<ItemList
				title={textGet("ai.summary.section.current_medications")}
				items={c.current_medications}
				sources={sourcesById}
			/>
			<ItemList
				title={textGet("ai.summary.section.relevant_exams")}
				items={c.relevant_exams}
				sources={sourcesById}
			/>
			<TextBlock
				title={textGet("ai.summary.section.background")}
				text={c.background}
			/>
			<TextBlock
				title={textGet("ai.summary.section.allergies")}
				text={c.allergies}
			/>
			<TextBlock
				title={textGet("ai.summary.section.recent_evolution")}
				text={c.recent_evolution}
			/>
			<p className="text-xs text-muted-foreground">
				{textGet("ai.summary.generated_at", {
					date: formatDateTime(summary.generated_at),
				})}
				{summary.omitted_consultations > 0 &&
					` · ${textGet("ai.summary.omitted", {
						count: summary.omitted_consultations,
					})}`}
			</p>
		</div>
	);
}

function TextBlock({ title, text }: { title: string; text: string }) {
	if (!text) return null;
	return (
		<div>
			<h3 className="text-xs font-medium uppercase text-muted-foreground">
				{title}
			</h3>
			<p>{text}</p>
		</div>
	);
}

function ItemList({
	title,
	items,
	sources,
	tone,
}: {
	title: string;
	items: AISummaryItem[] | null | undefined;
	sources: Map<number, AISummarySource>;
	tone?: "alert";
}) {
	const { textGet, formatDate } = useText();
	if (!items || items.length === 0) return null;
	return (
		<div>
			<h3 className="text-xs font-medium uppercase text-muted-foreground">
				{title}
			</h3>
			<ul className="grid gap-1">
				{items.map((item) => (
					<li
						key={item.text}
						className={
							tone === "alert" ? "text-amber-800 dark:text-amber-300" : ""
						}
					>
						{item.text}
						{item.sources.map((id) => {
							const src = sources.get(id);
							if (!src) return null;
							return (
								<Link
									key={id}
									to={`/clinical/medical-records/view/${id}`}
									className="ml-1.5 text-xs text-violet-700 underline-offset-2 hover:underline dark:text-violet-300"
									title={src.motive}
								>
									{textGet("ai.summary.source", {
										date: formatDate(src.date),
									})}
								</Link>
							);
						})}
					</li>
				))}
			</ul>
		</div>
	);
}
