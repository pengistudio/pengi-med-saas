import { useText } from "@pengi/shared";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
	Button,
	Textarea,
} from "@pengi/ui";
import {
	Check,
	ChevronDown,
	CircleAlert,
	Loader2,
	Plus,
	Sparkles,
} from "lucide-react";
import React from "react";
import {
	type Control,
	type FieldValues,
	type Path,
	useWatch,
} from "react-hook-form";
import {
	type AIDiagnosisSuggestion,
	type AIScribeDraft,
	type AIScribeVitals,
	type AIStatus,
	getAIStatus,
	organizeNotesWithAI,
} from "@/api/ai-service";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useEnabledFeatures } from "@/hooks/use-enabled-features";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { AI_CARD_CLASS, AIDraftTag, AIMark } from "./ai-mark";

export const SOAP_KEYS = [
	"subjective",
	"objective",
	"assessment",
	"plan",
] as const;
export type SoapKey = (typeof SOAP_KEYS)[number];
export type SoapValues = Partial<Record<SoapKey, string>>;

/** Form vital-sign values (possibly strings while typing) as scribe context. */
export function toScribeVitals(
	raw: Record<string, unknown> | null | undefined,
): AIScribeVitals | undefined {
	if (!raw) return undefined;
	const num = (v: unknown, int = false) => {
		const n = typeof v === "number" ? v : Number(String(v ?? "").trim());
		if (!Number.isFinite(n) || n <= 0 || String(v ?? "").trim() === "")
			return undefined;
		return int ? Math.round(n) : n;
	};
	const vitals: AIScribeVitals = {
		weight: num(raw.weight),
		height: num(raw.height),
		blood_pressure:
			typeof raw.blood_pressure === "string" && raw.blood_pressure.trim()
				? raw.blood_pressure.trim()
				: undefined,
		temperature: num(raw.temperature),
		heart_rate: num(raw.heart_rate, true),
		o2_saturation: num(raw.o2_saturation, true),
	};
	return Object.values(vitals).some((v) => v !== undefined)
		? vitals
		: undefined;
}

export interface DiagnosisValue {
	code: string;
	title: string;
}

/**
 * Whether the scribe may be offered: the plan includes AI, and the user holds
 * USE_AI_SCRIBE and CREATE_MEDICAL_RECORD. The clinic's opt-in and the
 * provider key come from the API (`AIStatus`).
 */
export function useCanUseAIScribe(): boolean {
	const planHasAI = useEnabledFeatures().ai === true;
	const { checkPermission } = usePermission();
	return (
		planHasAI &&
		checkPermission([
			PERMISSIONS.AI.PERMISSION_USE_AI_SCRIBE,
			PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
		])
	);
}

/**
 * Tracks which S/O/A/P sections hold text the scribe wrote (see
 * SoapDraftTag): the card reports what it applied, the tags compare it with
 * the form.
 */
export function useAIDraftSoap() {
	const [applied, setApplied] = React.useState<SoapValues>({});
	return {
		applied,
		record: (values: SoapValues) =>
			setApplied((prev) => ({ ...prev, ...values })),
	};
}

/**
 * "AI draft" tag for one S/O/A/P section. It shows only while the section's
 * text is exactly what the model produced: once the doctor edits it, it is
 * theirs and the tag goes away.
 */
export function SoapDraftTag<T extends FieldValues>({
	control,
	name,
	applied,
}: {
	control: Control<T>;
	name: SoapKey;
	applied: SoapValues;
}) {
	const value = useWatch({
		control,
		name: `soap_record.${name}` as Path<T>,
	});
	if (!applied[name] || String(value ?? "").trim() !== applied[name]) {
		return null;
	}
	return <AIDraftTag />;
}

interface AIScribeCardProps {
	patientId: number;
	/** Current S/O/A/P values of the form. */
	getSoap: () => SoapValues;
	/** Writes the chosen S/O/A/P values into the form. */
	applySoap: (values: SoapValues) => void;
	/** Context already typed in the form (no identifiers). */
	getContext?: () => { motive?: string; vitals?: AIScribeVitals };
	/** Diagnoses already in the form; chips for these show as added. */
	diagnoses?: DiagnosisValue[];
	/** Adds a suggested diagnosis; without it no chips are shown. */
	onAddDiagnosis?: (d: DiagnosisValue) => void;
	/** Called with the values written into the form (see useAIDraftSoap). */
	onApplied?: (values: SoapValues) => void;
}

/**
 * "Notas rápidas" + "Ordenar con IA": turns the doctor's raw notes into an
 * S/O/A/P draft. The notes live only in this component's state: they are
 * not part of the form (so not in the autosaved draft) and are never stored.
 * Non-empty fields are replaced only after a confirmation that lists them;
 * suggested diagnoses are added one by one with a click.
 */
export function AIScribeCard({
	patientId,
	getSoap,
	applySoap,
	getContext,
	diagnoses,
	onAddDiagnosis,
	onApplied,
}: AIScribeCardProps) {
	const allowed = useCanUseAIScribe();
	const { textGet } = useText();
	const [status, setStatus] = React.useState<AIStatus | null>(null);
	const [notes, setNotes] = React.useState("");
	const [loading, setLoading] = React.useState(false);
	const [error, setError] = React.useState<string | null>(null);
	const [pending, setPending] = React.useState<{
		values: SoapValues;
		replaced: SoapKey[];
	} | null>(null);
	const [suggestions, setSuggestions] = React.useState<AIDiagnosisSuggestion[]>(
		[],
	);
	const [applied, setApplied] = React.useState(false);
	const [open, setOpen] = React.useState(false);

	React.useEffect(() => {
		if (!allowed) return;
		let cancelled = false;
		getAIStatus().then((res) => {
			if (!cancelled && res.success) setStatus(res.data);
		});
		return () => {
			cancelled = true;
		};
	}, [allowed]);

	if (!allowed || !status?.enabled || status.functions?.scribe !== true) {
		return null;
	}

	const quota = status.quotas.find((q) => q.function === "scribe");

	// Literal keys, so the message-keys test checks them.
	const fieldLabel = (key: SoapKey) =>
		key === "subjective"
			? textGet("form.create_medical_record.soap.subjective.title")
			: key === "objective"
				? textGet("form.create_medical_record.soap.objective.title")
				: key === "assessment"
					? textGet("form.create_medical_record.soap.assessment.title")
					: textGet("form.create_medical_record.soap.plan.title");

	function apply(values: SoapValues) {
		applySoap(values);
		onApplied?.(values);
		setApplied(true);
		setPending(null);
	}

	async function handleOrganize() {
		setLoading(true);
		setError(null);
		const context = getContext?.() ?? {};
		const res = await organizeNotesWithAI({
			patient_id: patientId,
			notes,
			motive: context.motive,
			vitals: context.vitals,
		});
		setLoading(false);
		if (!res.success) {
			setError(res.data?.error_message || textGet("ai.scribe.error"));
			return;
		}
		const draft: AIScribeDraft = res.data;
		if (quota) {
			setStatus((s) =>
				s
					? {
							...s,
							quotas: s.quotas.map((q) =>
								q.function === "scribe" ? { ...q, used: q.used + 1 } : q,
							),
						}
					: s,
			);
		}
		setSuggestions(draft.suggested_diagnoses ?? []);
		// Only sections the draft fills; empty ones never clear the form.
		const values: SoapValues = {};
		for (const key of SOAP_KEYS) {
			if (draft[key]?.trim()) values[key] = draft[key].trim();
		}
		if (Object.keys(values).length === 0) {
			setError(textGet("ai.scribe.error.empty"));
			return;
		}
		const current = getSoap();
		const replaced = SOAP_KEYS.filter(
			(key) => values[key] && current[key]?.trim(),
		);
		if (replaced.length > 0) setPending({ values, replaced });
		else apply(values);
	}

	const added = new Set((diagnoses ?? []).map((d) => d.code));

	return (
		<section aria-label={textGet("ai.scribe.title")} className={AI_CARD_CLASS}>
			<button
				type="button"
				aria-expanded={open}
				aria-controls="ai-scribe-body"
				onClick={() => setOpen((v) => !v)}
				className="flex w-full items-center gap-3 rounded-lg p-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
			>
				<AIMark />
				<span className="min-w-0 flex-1">
					<span className="block text-sm font-semibold">
						{textGet("ai.scribe.title")}
					</span>
					<span className="block text-xs text-muted-foreground">
						{textGet("ai.scribe.subtitle")}
					</span>
				</span>
				<ChevronDown
					aria-hidden="true"
					className={cn(
						"size-4 shrink-0 text-muted-foreground transition-transform",
						open && "rotate-180",
					)}
				/>
			</button>

			{open && (
				<div id="ai-scribe-body" className="border-t px-4 pb-4 pt-3">
					<label htmlFor="ai-scribe-notes" className="text-sm font-medium">
						{textGet("ai.scribe.notes")}
					</label>
					<Textarea
						id="ai-scribe-notes"
						className="mt-1.5"
						rows={4}
						maxLength={6000}
						value={notes}
						placeholder={textGet("ai.scribe.notes.placeholder")}
						onChange={(e) => setNotes(e.target.value)}
					/>
					<div className="mt-2 flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
						{applied ? (
							<p className="flex items-center gap-1.5 text-xs font-medium text-violet-700 dark:text-violet-300">
								<Check className="size-3.5" aria-hidden="true" />
								{textGet("ai.scribe.applied")}
							</p>
						) : (
							<p className="text-xs text-muted-foreground">
								{textGet("ai.scribe.disclaimer")}
							</p>
						)}
						<div className="flex items-center gap-3">
							{quota && quota.limit >= 0 && (
								<span className="text-xs tabular-nums text-muted-foreground">
									{textGet("ai.summary.quota", {
										used: quota.used,
										limit: quota.limit,
									})}
								</span>
							)}
							<Button
								type="button"
								size="sm"
								disabled={loading || notes.trim() === ""}
								onClick={handleOrganize}
							>
								{loading ? <Loader2 className="animate-spin" /> : <Sparkles />}
								{loading
									? textGet("ai.scribe.organizing")
									: textGet("ai.scribe.organize")}
							</Button>
						</div>
					</div>

					{error && (
						<Alert variant="destructive" className="mt-3">
							<CircleAlert />
							<AlertDescription>{error}</AlertDescription>
						</Alert>
					)}

					{onAddDiagnosis && suggestions.length > 0 && (
						<div className="mt-3 grid gap-1.5">
							<p className="text-xs font-medium text-muted-foreground">
								{textGet("ai.scribe.suggested_diagnoses")}
							</p>
							<div className="flex flex-wrap gap-1.5">
								{suggestions.map((s) => {
									const isAdded = added.has(s.code);
									return (
										<Button
											key={s.code}
											type="button"
											size="sm"
											variant="outline"
											disabled={isAdded}
											title={s.term}
											onClick={() =>
												onAddDiagnosis({ code: s.code, title: s.title })
											}
										>
											{isAdded ? <Check /> : <Plus />}
											<span className="font-mono text-xs">{s.code}</span>
											<span className="truncate">{s.title}</span>
											{isAdded && (
												<span className="sr-only">
													{textGet("ai.scribe.diagnosis_added")}
												</span>
											)}
										</Button>
									);
								})}
							</div>
						</div>
					)}
				</div>
			)}

			<AlertDialog open={pending !== null}>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							{textGet("ai.scribe.confirm.title")}
						</AlertDialogTitle>
						<AlertDialogDescription>
							{textGet("ai.scribe.confirm.description", {
								fields: (pending?.replaced ?? []).map(fieldLabel).join(", "),
							})}
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel
							onClick={() => {
								// Keep what the doctor wrote: fill only the empty fields.
								if (!pending) return;
								const onlyEmpty: SoapValues = {};
								for (const key of SOAP_KEYS) {
									if (pending.values[key] && !pending.replaced.includes(key))
										onlyEmpty[key] = pending.values[key];
								}
								if (Object.keys(onlyEmpty).length > 0) apply(onlyEmpty);
								else setPending(null);
							}}
						>
							{textGet("ai.scribe.confirm.keep")}
						</AlertDialogCancel>
						<AlertDialogAction onClick={() => pending && apply(pending.values)}>
							{textGet("ai.scribe.confirm.replace")}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</section>
	);
}
