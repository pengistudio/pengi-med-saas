import {
	type BaseModel,
	createHttpService,
	type ServiceResponse,
} from "@pengi/shared";
import { apiWithTenant } from ".";

const attachmentService = createHttpService(apiWithTenant);

/** Fixed attachment categories (mirror of the backend's AttachmentCategories). */
export const ATTACHMENT_CATEGORIES = [
	"lab_result",
	"imaging",
	"external_report",
	"clinical_photo",
	"consent",
	"other",
] as const;

export type AttachmentCategory = (typeof ATTACHMENT_CATEGORIES)[number];

/** Largest file the server accepts (15 MB). */
export const MAX_ATTACHMENT_SIZE = 15 * 1024 * 1024;

/** A file attached to a patient (Adjunto). */
export interface PatientAttachment extends BaseModel {
	tenant_id: number;
	patient_id: number;
	medical_record_id?: number | null;
	/** Date of the linked consultation; only set by the list. */
	medical_record_date?: string | null;
	/** Result of a reviewed exam: it can't be deleted. Only set by the list. */
	locked_by_review?: boolean;
	category: AttachmentCategory;
	/** Exam day, a calendar date ("2026-03-15T00:00:00Z"). */
	taken_at: string;
	description: string;
	file_name: string;
	mime_type: string;
	size: number;
	sha256: string;
	uploaded_by_id: number;
}

/** A deleted attachment, with who deleted it and why (DeletedAt is when). */
export interface DeletedPatientAttachment extends PatientAttachment {
	deleted_by_id?: number | null;
	deleted_by_name?: string;
	delete_reason?: string;
}

/** Backend error code when the plan's storage quota is used up. */
export const STORAGE_QUOTA_ERROR_CODE = "E-PLAN-003";

/** Set on an upload whose content already exists for the patient (not blocking). */
export interface AttachmentDuplicate {
	id: number;
	file_name: string;
	created_at: string;
}

/** The upload response: the stored attachment plus an optional duplicate warning. */
export interface UploadedAttachment extends PatientAttachment {
	duplicate_of?: AttachmentDuplicate;
}

export interface UploadAttachmentPayload {
	file: File;
	category: AttachmentCategory;
	/** "YYYY-MM-DD"; the server uses the upload day when omitted. */
	taken_at?: string;
	description?: string;
	/** Consultation (medical record) of the same patient to link it to. */
	medical_record_id?: number;
}

const base = (patientId: number) =>
	`/clinical/patients/${patientId}/attachments`;

/** The tenant's attachment storage against its plan quota. */
export interface AttachmentUsage {
	used_bytes: number;
	/** 0 = the plan allows no attachments. */
	quota_bytes: number;
	/** Set from 80% of the quota on. */
	warning: boolean;
}

/** The list response: the patient's attachments plus the tenant's usage. */
export interface PatientAttachmentList {
	items: PatientAttachment[];
	usage: AttachmentUsage;
}

/** Whether the quota is used up: no upload can fit. */
export const isStorageFull = (usage: AttachmentUsage) =>
	usage.used_bytes >= usage.quota_bytes;

export const getPatientAttachments = async (
	patientId: number,
	category?: AttachmentCategory,
	/** Only the attachments linked to this consultation. */
	medicalRecordId?: number,
): Promise<ServiceResponse<PatientAttachmentList>> => {
	const params: Record<string, string | number> = {};
	if (category) params.category = category;
	if (medicalRecordId) params.medical_record_id = medicalRecordId;
	return attachmentService.get<PatientAttachmentList>(base(patientId), {
		params: Object.keys(params).length > 0 ? params : undefined,
		notifyError: true,
	});
};

export const uploadPatientAttachment = async (
	patientId: number,
	payload: UploadAttachmentPayload,
	options: {
		/** 0-100 of the request body sent. */
		onProgress?: (percent: number) => void;
		/** Batch uploads report in the UI instead of one toast per file. */
		notify?: boolean;
	} = {},
): Promise<ServiceResponse<UploadedAttachment>> => {
	const formData = new FormData();
	formData.append("file", payload.file);
	formData.append("category", payload.category);
	if (payload.taken_at) formData.append("taken_at", payload.taken_at);
	if (payload.description) formData.append("description", payload.description);
	if (payload.medical_record_id)
		formData.append("medical_record_id", String(payload.medical_record_id));
	const notify = options.notify ?? true;
	return attachmentService.postForm<UploadedAttachment>(
		base(patientId),
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

export type UpdateAttachmentPayload = {
	category: AttachmentCategory;
	/** "YYYY-MM-DD". */
	taken_at: string;
	description: string;
};

/** Edits the metadata only; the file never changes. */
export const updatePatientAttachment = async (
	patientId: number,
	attachmentId: number,
	payload: UpdateAttachmentPayload,
): Promise<ServiceResponse<PatientAttachment>> =>
	attachmentService.put<PatientAttachment>(
		`${base(patientId)}/${attachmentId}`,
		payload,
		{ notifySuccess: true, notifyError: true },
	);

/**
 * Links an attachment to a consultation of the same patient, or unlinks it
 * (null). Batch linking after a consultation is created passes notifySuccess
 * false so it doesn't toast once per file.
 */
export const linkPatientAttachment = async (
	patientId: number,
	attachmentId: number,
	medicalRecordId: number | null,
	options: { notifySuccess?: boolean } = {},
): Promise<ServiceResponse<PatientAttachment>> =>
	attachmentService.put<PatientAttachment>(
		`${base(patientId)}/${attachmentId}`,
		{ medical_record_id: medicalRecordId },
		{ notifySuccess: options.notifySuccess ?? true, notifyError: true },
	);

export const downloadPatientAttachment = async (
	patientId: number,
	attachmentId: number,
): Promise<ServiceResponse<Blob>> =>
	attachmentService.get<Blob>(`${base(patientId)}/${attachmentId}/download`, {
		responseType: "blob",
		notifyError: true,
	});

/** The file's bytes for the in-app viewer (audited server-side as a view). */
export const viewPatientAttachment = async (
	patientId: number,
	attachmentId: number,
): Promise<ServiceResponse<Blob>> =>
	attachmentService.get<Blob>(`${base(patientId)}/${attachmentId}/view`, {
		responseType: "blob",
		notifyError: true,
	});

/** Soft-deletes an attachment; the reason is required by the server. */
export const deletePatientAttachment = async (
	patientId: number,
	attachmentId: number,
	reason: string,
): Promise<ServiceResponse<null>> =>
	attachmentService.delete<null>(`${base(patientId)}/${attachmentId}`, {
		data: { reason },
		notifySuccess: true,
		notifyError: true,
	});

export const getDeletedPatientAttachments = async (
	patientId: number,
): Promise<ServiceResponse<DeletedPatientAttachment[]>> =>
	attachmentService.get<DeletedPatientAttachment[]>(
		`/clinical/patients/${patientId}/attachments-deleted`,
		{ notifyError: true },
	);

export const restorePatientAttachment = async (
	patientId: number,
	attachmentId: number,
): Promise<ServiceResponse<null>> =>
	attachmentService.post<null>(
		`${base(patientId)}/${attachmentId}/restore`,
		undefined,
		{ notifySuccess: true, notifyError: true },
	);
