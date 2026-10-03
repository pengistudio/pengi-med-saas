import { MAX_ATTACHMENT_SIZE } from "@/api/patient-attachment-service";
import { isHeicFile } from "@/lib/heic";

/** What the file picker offers (HEIC is converted to JPEG before upload). */
export const ATTACHMENT_ACCEPT =
	"application/pdf,image/jpeg,image/png,image/heic,image/heif,.heic,.heif";

const ACCEPTED_TYPES = ["application/pdf", "image/jpeg", "image/png"];
const ACCEPTED_EXTENSION = /\.(pdf|jpe?g|png)$/i;

/** i18n key of why a file can't be uploaded, or null when it can. */
export function attachmentFileError(file: File): string | null {
	if (file.size > MAX_ATTACHMENT_SIZE)
		return "clinical.attachment.form.error.too_large";
	if (
		!isHeicFile(file) &&
		!ACCEPTED_TYPES.includes(file.type) &&
		!ACCEPTED_EXTENSION.test(file.name)
	)
		return "clinical.attachment.form.error.type";
	return null;
}
