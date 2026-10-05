import {
	type ExamResultUpload,
	uploadExamResult,
} from "@/api/exam-order-service";
import type { AttachmentCategory } from "@/api/patient-attachment-service";
import {
	type GenericBatchResult,
	runUploadBatch,
	type UploadItem,
} from "@/components/features/patient-attachments/upload-batch";

export interface ExamResultsMeta {
	/** The order's exams every file of the batch covers. */
	item_ids: number[];
	category?: AttachmentCategory;
	taken_at?: string;
	description?: string;
}

/**
 * Uploads result files of an order: one request per file, each with the
 * selected exams, sharing the attachments' batch rules (validation, HEIC
 * conversion, quota stop, duplicate warning).
 */
export function uploadExamResults(
	orderId: number,
	files: File[],
	meta: ExamResultsMeta,
	onUpdate: (id: number, patch: Partial<UploadItem>) => void,
): Promise<GenericBatchResult<ExamResultUpload>> {
	return runUploadBatch<ExamResultUpload>(
		files,
		async (file, onProgress) => {
			const res = await uploadExamResult(
				orderId,
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
}
