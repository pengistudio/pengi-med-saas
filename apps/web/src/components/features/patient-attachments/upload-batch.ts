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
	const result: BatchResult = {
		uploaded: 0,
		failed: 0,
		quotaReached: false,
		attachments: [],
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
		const res = await uploadPatientAttachment(
			patientId,
			{ file, ...meta },
			{ notify: false, onProgress: (progress) => onUpdate(id, { progress }) },
		);
		if (res.success) {
			result.uploaded++;
			if (res.data) result.attachments.push(res.data);
			const duplicate = res.data?.duplicate_of;
			onUpdate(id, {
				status: duplicate ? "duplicate" : "done",
				progress: 100,
				duplicate,
			});
		} else {
			result.failed++;
			if (res.data?.error_code === STORAGE_QUOTA_ERROR_CODE)
				result.quotaReached = true;
			onUpdate(id, { status: "error", errorMessage: res.message });
		}
	}
	return result;
}
