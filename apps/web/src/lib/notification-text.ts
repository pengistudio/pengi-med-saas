import type { AppText } from "@pengi/shared";
import type { Notification } from "@/api/notification-service";
import { notificationTextParams } from "@/lib/whatsapp-alerts";

const WHATSAPP_MESSAGE_KEY = "notification.whatsapp.message";
/** Content types a WhatsApp message notification may carry without text. */
const WHATSAPP_MEDIA_TYPES = new Set(["image", "audio", "document", "other"]);

/**
 * When the draft behind a stale-draft notification was last saved: its
 * `draft_updated_at`, or for older notifications, which only carry
 * `minutes_elapsed` at the time they were created, that many minutes before
 * the notification.
 */
function draftUpdatedAt(
	notification: Pick<Notification, "params"> &
		Partial<Pick<Notification, "CreatedAt">>,
): Date | string | undefined {
	const { draft_updated_at, minutes_elapsed } = notification.params;
	if (draft_updated_at) return draft_updated_at;
	const minutes = Number(minutes_elapsed);
	if (!notification.CreatedAt || !minutes_elapsed || Number.isNaN(minutes))
		return undefined;
	return new Date(
		new Date(notification.CreatedAt).getTime() - minutes * 60_000,
	);
}

/**
 * Resolves a notification's full display text: its i18n template filled with
 * its params, computing the live `{elapsed}` value for stale-draft
 * notifications.
 */
export function getNotificationText(
	notification: Pick<Notification, "message_key" | "params"> &
		Partial<Pick<Notification, "CreatedAt">>,
	{ textGet, formatRelative }: Pick<AppText, "textGet" | "formatRelative">,
): string {
	const updatedAt = draftUpdatedAt(notification);
	// Numeric `count` picks the plural form (WhatsApp message notifications).
	const params = notificationTextParams(notification.params);
	if (updatedAt) {
		// The template already says "desde hace" / "ago".
		params.elapsed = formatRelative(updatedAt, { suffix: false });
	}
	// A photo or audio has no text: describe it instead of an empty preview.
	const contentType = String(params.content_type ?? "");
	if (
		notification.message_key === WHATSAPP_MESSAGE_KEY &&
		!params.preview &&
		WHATSAPP_MEDIA_TYPES.has(contentType)
	) {
		params.preview = textGet(`whatsapp.inbox.content.${contentType}`);
	}
	return textGet(notification.message_key, params);
}
