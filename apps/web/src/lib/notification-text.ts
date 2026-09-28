import type { AppText } from "@pengi/shared";
import type { Notification } from "@/api/notification-service";

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
	const params = updatedAt
		? {
				...notification.params,
				// The template already says "desde hace" / "ago".
				elapsed: formatRelative(updatedAt, { suffix: false }),
			}
		: notification.params;
	return textGet(notification.message_key, params);
}
