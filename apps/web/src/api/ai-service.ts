import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from ".";

const aiService = createHttpService(apiWithTenant);

/** A tenant's uses of one AI function this month (`limit` -1 = unlimited). */
export interface AIQuota {
	function: "summary" | "scribe" | "bot";
	used: number;
	limit: number;
	period_start: string;
	period_end: string;
}

export interface AIStatus {
	/** The summary has a provider key (backoffice or AI_API_KEY). */
	configured: boolean;
	/** The clinic opted in (clinic settings). */
	enabled: boolean;
	quotas: AIQuota[];
	/** Whether each function has a provider key. Absent in older APIs. */
	functions?: Partial<Record<AIQuota["function"], boolean>>;
}

/** One statement of the summary and the consultations (record ids) it cites. */
export interface AISummaryItem {
	text: string;
	sources: number[];
}

export interface AISummaryContent {
	overview: string;
	active_problems: AISummaryItem[];
	background: string;
	allergies: string;
	current_medications: AISummaryItem[];
	relevant_exams: AISummaryItem[];
	recent_evolution: string;
	alerts: AISummaryItem[];
}

export interface AISummarySource {
	label: string;
	medical_record_id: number;
	date: string;
	motive: string;
}

/** An AI draft: never part of the medical record, not printed or signed. */
export interface AISummary {
	patient_id: number;
	content: AISummaryContent;
	sources: AISummarySource[];
	included_consultations: number;
	omitted_consultations: number;
	model: string;
	generated_at: string;
	cached: boolean;
	ai_draft: true;
}

export const getAIStatus = async (): Promise<ServiceResponse<AIStatus>> =>
	aiService.get<AIStatus>("/ai/status");

/** The cached summary if the history didn't change since, else null. Free. */
export const getPatientAISummary = async (
	patientId: number,
): Promise<ServiceResponse<AISummary | null>> =>
	aiService.get<AISummary | null>(`/ai/patients/${patientId}/summary`);

/**
 * Generates the summary (one use of the monthly quota) or returns the cached
 * one. Errors (not configured, quota, provider) are shown in the panel.
 */
export const generatePatientAISummary = async (
	patientId: number,
): Promise<ServiceResponse<AISummary>> =>
	aiService.post<AISummary>(`/ai/patients/${patientId}/summary`, undefined, {
		notifyError: false,
	});

/** Vital signs already typed in the consultation form (context only). */
export interface AIScribeVitals {
	weight?: number | null;
	height?: number | null;
	blood_pressure?: string;
	temperature?: number | null;
	heart_rate?: number | null;
	o2_saturation?: number | null;
}

/** One "Ordenar con IA": the notes are sent once and never stored. */
export interface AIScribeRequest {
	patient_id: number;
	notes: string;
	motive?: string;
	vitals?: AIScribeVitals;
}

/** A diagnosis code that exists in the clinic's catalog (picker shape). */
export interface AIDiagnosisSuggestion {
	term: string;
	code: string;
	title: string;
}

/** AI draft of the SOAP fields: the doctor reviews it before saving. */
export interface AIScribeDraft {
	subjective: string;
	objective: string;
	assessment: string;
	plan: string;
	suggested_diagnoses: AIDiagnosisSuggestion[];
	diagnosis_system: "cie10" | "cie11";
	model: string;
	ai_draft: true;
}

/**
 * Orders the notes into a SOAP draft (one "scribe" use of the monthly
 * quota). Errors (not configured, quota, provider) are shown inline.
 */
export const organizeNotesWithAI = async (
	payload: AIScribeRequest,
): Promise<ServiceResponse<AIScribeDraft>> =>
	aiService.post<AIScribeDraft>(
		"/ai/scribe",
		payload as unknown as Record<string, unknown>,
		{ notifyError: false },
	);
