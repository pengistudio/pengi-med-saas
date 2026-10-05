import {
	type BaseModel,
	createHttpService,
	type ServiceResponse,
} from "@pengi/shared";
import { apiWithTenant } from ".";
import type {
	DiagnosisItem,
	PaginatedResponse,
	Patient,
} from "./clinical-service";
import type {
	AttachmentCategory,
	AttachmentDuplicate,
	PatientAttachment,
} from "./patient-attachment-service";
import type { DocumentSignature } from "./signature-service";

const examOrderService = createHttpService(apiWithTenant);

/** Exam categories (mirror of the backend's clinical_data.ExamCategory*). */
export const EXAM_CATEGORIES = ["laboratory", "imaging", "other"] as const;
export type ExamCategory = (typeof EXAM_CATEGORIES)[number];

/** Order statuses; the server recomputes them from the results. */
export const EXAM_ORDER_STATUSES = [
	"issued",
	"partial_results",
	"complete_results",
	"voided",
] as const;
export type ExamOrderStatus = (typeof EXAM_ORDER_STATUSES)[number];

export const EXAM_PRIORITIES = ["routine", "urgent"] as const;
export type ExamPriority = (typeof EXAM_PRIORITIES)[number];

/** One exam a tenant can order (catálogo de exámenes). */
export interface ExamCatalogItem extends BaseModel {
	tenant_id: number;
	name: string;
	category: ExamCategory;
	subgroup: string;
	default_indications: string;
	active: boolean;
	is_default: boolean;
	sort_order: number;
}

/** A named set of catalog exams ordered together (perfil). */
export interface ExamProfile extends BaseModel {
	tenant_id: number;
	name: string;
	active: boolean;
	is_default: boolean;
	items: ExamCatalogItem[];
}

/** One exam of an order; `attachments` are its results (Adjuntos). */
export interface ExamOrderItem extends BaseModel {
	tenant_id: number;
	exam_order_id: number;
	catalog_item_id: number | null;
	name: string;
	category: ExamCategory;
	subgroup: string;
	indications: string;
	reviewed_by_id: number | null;
	reviewed_at: string | null;
	/** Who reviewed it; absent until the item is reviewed. */
	reviewed_by_name?: string;
	attachments: PatientAttachment[];
}

export interface ExamOrder extends BaseModel, DocumentSignature {
	tenant_id: number;
	number: number;
	/** "ORD-000123", filled by the server. */
	code: string;
	patient_id: number;
	patient?: Patient;
	medical_record_id: number | null;
	ordered_by_id: number;
	ordered_by_name: string;
	diagnoses: DiagnosisItem[] | null;
	priority: ExamPriority;
	notes: string;
	destination_lab: string;
	status: ExamOrderStatus;
	closed_manually: boolean;
	void_reason: string;
	voided_at: string | null;
	voided_by_id: number | null;
	/** Some exam has a result nobody reviewed yet. */
	pending_review: boolean;
	items: ExamOrderItem[];
}

/**
 * One exam sent on create/update. `id` = an existing item (update only);
 * with `catalog_item_id` the server copies name/category/subgroup from the
 * catalog; without it `name` is required. `indications` undefined = the
 * catalog's default (new item) or unchanged (existing).
 */
export interface ExamOrderItemInput {
	id?: number;
	catalog_item_id?: number;
	name?: string;
	category?: ExamCategory;
	subgroup?: string;
	indications?: string;
}

export type ExamOrderHeaderPayload = {
	medical_record_id?: number | null;
	diagnoses: DiagnosisItem[];
	priority: ExamPriority;
	notes: string;
	destination_lab: string;
	items: ExamOrderItemInput[];
};

export type CreateExamOrderPayload = ExamOrderHeaderPayload & {
	patient_id: number;
};

export type UpdateExamOrderPayload = ExamOrderHeaderPayload;

export interface ExamOrderFilters {
	/** One or more statuses (sent comma-separated). */
	status?: ExamOrderStatus[];
	pending_review?: boolean;
	/** "me" or a user id. */
	ordered_by?: "me" | number;
	patient_id?: number;
	record_id?: number;
	page?: number;
	limit?: number;
}

/** Query params for the list endpoint; empty filters are left out. */
export function examOrderQuery(
	filters: ExamOrderFilters,
): Record<string, string | number> {
	const params: Record<string, string | number> = {};
	if (filters.status && filters.status.length > 0)
		params.status = filters.status.join(",");
	if (filters.pending_review) params.pending_review = "true";
	if (filters.ordered_by !== undefined) params.ordered_by = filters.ordered_by;
	if (filters.patient_id) params.patient_id = filters.patient_id;
	if (filters.record_id) params.record_id = filters.record_id;
	if (filters.page) params.page = filters.page;
	if (filters.limit) params.limit = filters.limit;
	return params;
}

// ─── Catalog ────────────────────────────────────────────────────────────────

export interface ExamCatalogFilters {
	active?: boolean;
	category?: ExamCategory;
	q?: string;
}

export type ExamCatalogItemPayload = {
	name: string;
	category: ExamCategory;
	subgroup: string;
	default_indications: string;
};

export const getExamCatalog = async (
	filters: ExamCatalogFilters = {},
): Promise<ServiceResponse<ExamCatalogItem[]>> => {
	const params: Record<string, string> = {};
	if (filters.active !== undefined) params.active = String(filters.active);
	if (filters.category) params.category = filters.category;
	if (filters.q?.trim()) params.q = filters.q.trim();
	return examOrderService.get<ExamCatalogItem[]>("/clinical/exam-catalog", {
		params: Object.keys(params).length > 0 ? params : undefined,
		notifyError: true,
	});
};

export const createExamCatalogItem = async (payload: ExamCatalogItemPayload) =>
	examOrderService.post<ExamCatalogItem>("/clinical/exam-catalog", payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateExamCatalogItem = async (
	id: number,
	payload: Partial<ExamCatalogItemPayload> & { active?: boolean },
) =>
	examOrderService.put<ExamCatalogItem>(
		`/clinical/exam-catalog/${id}`,
		payload,
		{ notifySuccess: true, notifyError: true },
	);

export const deleteExamCatalogItem = async (id: number) =>
	examOrderService.delete<null>(`/clinical/exam-catalog/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});

export interface RestoreExamCatalogResult {
	restored_items: number;
	restored_profiles: number;
}

/** Re-activates (or re-creates) the seeded exams and profiles. */
export const restoreExamCatalogDefaults = async () =>
	examOrderService.post<RestoreExamCatalogResult>(
		"/clinical/exam-catalog/restore-defaults",
		undefined,
		{ notifySuccess: true, notifyError: true },
	);

export type ExamProfilePayload = {
	name: string;
	item_ids: number[];
};

export const getExamProfiles = async (
	filters: { active?: boolean } = {},
): Promise<ServiceResponse<ExamProfile[]>> =>
	examOrderService.get<ExamProfile[]>("/clinical/exam-profiles", {
		params:
			filters.active !== undefined
				? { active: String(filters.active) }
				: undefined,
		notifyError: true,
	});

export const createExamProfile = async (payload: ExamProfilePayload) =>
	examOrderService.post<ExamProfile>("/clinical/exam-profiles", payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateExamProfile = async (
	id: number,
	payload: Partial<ExamProfilePayload> & { active?: boolean },
) =>
	examOrderService.put<ExamProfile>(`/clinical/exam-profiles/${id}`, payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const deleteExamProfile = async (id: number) =>
	examOrderService.delete<null>(`/clinical/exam-profiles/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});

// ─── Orders ─────────────────────────────────────────────────────────────────

export const getExamOrders = async (
	filters: ExamOrderFilters = {},
): Promise<ServiceResponse<PaginatedResponse<ExamOrder>>> =>
	examOrderService.get<PaginatedResponse<ExamOrder>>("/clinical/exam-orders", {
		params: examOrderQuery(filters),
		notifyError: true,
	});

/** Orders of the current user with results waiting for review; silent (no toasts). */
export const getPendingReviewCount = async (): Promise<number> => {
	const res = await examOrderService.get<PaginatedResponse<ExamOrder>>(
		"/clinical/exam-orders",
		{
			params: examOrderQuery({
				pending_review: true,
				ordered_by: "me",
				limit: 1,
			}),
			notifyError: false,
		},
	);
	return res.success ? res.data.total : 0;
};

export const getExamOrder = async (
	id: number,
): Promise<ServiceResponse<ExamOrder>> =>
	examOrderService.get<ExamOrder>(`/clinical/exam-orders/${id}`, {
		notifyError: true,
	});

export const createExamOrder = async (payload: CreateExamOrderPayload) =>
	examOrderService.post<ExamOrder>("/clinical/exam-orders", payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateExamOrder = async (
	id: number,
	payload: UpdateExamOrderPayload,
) =>
	examOrderService.put<ExamOrder>(`/clinical/exam-orders/${id}`, payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const voidExamOrder = async (id: number, reason: string) =>
	examOrderService.post<ExamOrder>(
		`/clinical/exam-orders/${id}/void`,
		{ reason },
		{ notifySuccess: true, notifyError: true },
	);

/** Marks the order complete by hand (e.g. results arrived on paper). */
export const closeExamOrder = async (id: number) =>
	examOrderService.post<ExamOrder>(
		`/clinical/exam-orders/${id}/close`,
		undefined,
		{ notifySuccess: true, notifyError: true },
	);

export const downloadExamOrderPdf = async (
	id: number,
): Promise<ServiceResponse<Blob>> =>
	examOrderService.get<Blob>(`/clinical/exam-orders/${id}/download`, {
		responseType: "blob",
		notifyError: true,
	});

export const emailExamOrder = async (id: number, email: string) =>
	examOrderService.post<null>(
		`/clinical/exam-orders/${id}/email`,
		{ email },
		{ notifySuccess: true, notifyError: true },
	);

export const signExamOrder = async (id: number) =>
	examOrderService.post<ExamOrder>(
		`/clinical/exam-orders/${id}/sign`,
		undefined,
		{ notifySuccess: true, notifyError: true },
	);

// ─── Results ────────────────────────────────────────────────────────────────

export interface UploadExamResultPayload {
	file: File;
	/** The order's exams this file covers (at least one). */
	item_ids: number[];
	category?: AttachmentCategory;
	/** "YYYY-MM-DD"; the server uses the upload day when omitted. */
	taken_at?: string;
	description?: string;
}

/** The upload response: the order (status recomputed) and the new attachment. */
export interface ExamResultUpload {
	order: ExamOrder;
	attachment: PatientAttachment;
	duplicate_of?: AttachmentDuplicate;
}

/** Uploads ONE result file covering `item_ids` (one request per file). */
export const uploadExamResult = async (
	orderId: number,
	payload: UploadExamResultPayload,
	options: {
		onProgress?: (percent: number) => void;
		/** Batch uploads report in the UI instead of one toast per file. */
		notify?: boolean;
	} = {},
): Promise<ServiceResponse<ExamResultUpload>> => {
	const formData = new FormData();
	formData.append("file", payload.file);
	for (const id of payload.item_ids) formData.append("item_ids", String(id));
	if (payload.category) formData.append("category", payload.category);
	if (payload.taken_at) formData.append("taken_at", payload.taken_at);
	if (payload.description) formData.append("description", payload.description);
	const notify = options.notify ?? true;
	return examOrderService.postForm<ExamResultUpload>(
		`/clinical/exam-orders/${orderId}/results`,
		formData,
		{
			notifySuccess: notify,
			notifyError: notify,
			onUploadProgress: (event) => {
				if (event.total)
					options.onProgress?.(Math.round((event.loaded / event.total) * 100));
			},
		},
	);
};

/** Deletes a result (the attachment); the reason is required by the server. */
export const deleteExamResult = async (
	orderId: number,
	attachmentId: number,
	reason: string,
) =>
	examOrderService.delete<ExamOrder>(
		`/clinical/exam-orders/${orderId}/results/${attachmentId}`,
		{ data: { reason }, notifySuccess: true, notifyError: true },
	);

/** Marks the given exams' results as reviewed by the current user. */
export const reviewExamResults = async (orderId: number, itemIds: number[]) =>
	examOrderService.post<ExamOrder>(
		`/clinical/exam-orders/${orderId}/review`,
		{ item_ids: itemIds },
		{ notifySuccess: true, notifyError: true },
	);
