import {
	type AttachmentCategory,
	type AttachmentDuplicate,
	STORAGE_QUOTA_ERROR_CODE,
	type UploadedAttachment,
	uploadPatientAttachment,
} from "@/api/patient-attachment-service";
import { attachmentFileError } from "@/lib/attachment-files";
import { convertHeicToJpeg, isHeicFile } from "@/lib/heic";

export type UploadItemStatus =
	| "pending"
	| "converting"
	| "uploading"
	| "done"
	| "duplicate"
	| "error"
	| "skipped";

export interface UploadItem {
	id: number;
	name: string;
	status: UploadItemStatus;
	progress: number;
	/** i18n key (client validation) or a server message already translated. */
	errorKey?: string;
	errorMessage?: string;
	duplicate?: AttachmentDuplicate;
}

export interface BatchMeta {
	category: AttachmentCategory;
	taken_at?: string;
	description?: string;
	/** Links every file to this consultation of the same patient. */
	medical_record_id?: number;
}

export interface BatchResult {
	uploaded: number;
	failed: number;
	quotaReached: boolean;
	/** The stored attachments, in upload order. */
	attachments: UploadedAttachment[];
}

/** What one file's upload returns: success, a message and the duplicate warning. */
export interface UploadOneResult<T> {
	success: boolean;
	message?: string;
	data?: T;
	errorCode?: string;
	duplicate?: AttachmentDuplicate;
}

export interface GenericBatchResult<T> {
	uploaded: number;
	failed: number;
	quotaReached: boolean;
	/** What each successful upload returned, in upload order. */
	results: T[];
}

/**
 * Runs `upload` for each file, one request each, one after another. A failure
 * doesn't stop the others, except the plan's storage quota: then the rest is
 * skipped. Invalid files are rejected without a request; HEIC files are
 * converted to JPEG first. Shared by the patient attachments and the exam
 * results uploads.
 */
export async function runUploadBatch<T>(
	files: File[],
	upload: (
		file: File,
		onProgress: (percent: number) => void,
	) => Promise<UploadOneResult<T>>,
	onUpdate: (id: number, patch: Partial<UploadItem>) => void,
): Promise<GenericBatchResult<T>> {
	const result: GenericBatchResult<T> = {
		uploaded: 0,
		failed: 0,
		quotaReached: false,
		results: [],
	};
	for (let id = 0; id < files.length; id++) {
		if (result.quotaReached) {
			onUpdate(id, { status: "skipped" });
			continue;
		}
		let file = files[id];
		const invalid = attachmentFileError(file);
		if (invalid) {
			onUpdate(id, { status: "error", errorKey: invalid });
			result.failed++;
			continue;
		}
		if (isHeicFile(file)) {
			onUpdate(id, { status: "converting" });
			try {
				file = await convertHeicToJpeg(file);
			} catch {
				onUpdate(id, {
					status: "error",
					errorKey: "clinical.attachment.upload.error.convert",
				});
				result.failed++;
				continue;
			}
		}
		onUpdate(id, { status: "uploading", progress: 0 });
		const res = await upload(file, (progress) => onUpdate(id, { progress }));
		if (res.success) {
			result.uploaded++;
			if (res.data !== undefined) result.results.push(res.data);
			onUpdate(id, {
				status: res.duplicate ? "duplicate" : "done",
				progress: 100,
				duplicate: res.duplicate,
			});
		} else {
			result.failed++;
			if (res.errorCode === STORAGE_QUOTA_ERROR_CODE)
				result.quotaReached = true;
			onUpdate(id, { status: "error", errorMessage: res.message });
		}
	}
	return result;
}

/**
 * Uploads the files one request each, one after another, sharing the metadata.
 * A failure doesn't stop the others, except the plan's storage quota: then the
 * rest is skipped. HEIC files are converted to JPEG first.
 */
export async function uploadBatch(
	patientId: number,
	files: File[],
	meta: BatchMeta,
	onUpdate: (id: number, patch: Partial<UploadItem>) => void,
): Promise<BatchResult> {
	const { results, ...counts } = await runUploadBatch<UploadedAttachment>(
		files,
		async (file, onProgress) => {
			const res = await uploadPatientAttachment(
				patientId,
				{ file, ...meta },
				{ notify: false, onProgress },
			);
			return res.success
				? { success: true, data: res.data, duplicate: res.data?.duplicate_of }
				: {
						success: false,
						message: res.message,
						errorCode: res.data?.error_code,
					};
		},
		onUpdate,
	);
	return { ...counts, attachments: results };
}
