import type { AppText } from "@pengi/shared";
import { toast } from "sonner";
import {
	downloadPatientAttachment,
	type PatientAttachment,
} from "@/api/patient-attachment-service";
import type { BatchResult } from "./upload-batch";

/**
 * One summary toast for a whole upload batch instead of one per file: it
 * describes several API results, so it lives here; per-file errors show in the
 * progress list.
 */
export function toastBatchResult(
	result: Pick<BatchResult, "uploaded" | "failed" | "quotaReached">,
	textGet: AppText["textGet"],
) {
	if (result.quotaReached) {
		toast.error(textGet("plan.limit.storage"));
	} else if (result.failed === 0) {
		toast.success(
			textGet("clinical.attachment.upload.summary.success", {
				count: result.uploaded,
			}),
		);
	} else {
		toast.warning(
			textGet("clinical.attachment.upload.summary.partial", {
				uploaded: result.uploaded,
				failed: result.failed,
			}),
		);
	}
}

/** Downloads the original file and hands it to the browser as a save. */
export async function saveAttachmentFile(
	patientId: number,
	attachment: PatientAttachment,
) {
	const res = await downloadPatientAttachment(patientId, attachment.ID);
	if (!res.success || !res.data) return;
	const url = window.URL.createObjectURL(res.data);
	const link = document.createElement("a");
	link.href = url;
	link.download = attachment.file_name;
	link.click();
	window.URL.revokeObjectURL(url);
}
