import { formatDistanceToNow } from "date-fns";
import { enUS, es } from "date-fns/locale";

/**
 * Fills a `{{placeholder}}` template (an i18n string resolved via textGet)
 * with a notification's params. Kept separate from useText/textGet since
 * that hook is a plain key->string lookup with no interpolation support —
 * this is the one place today that needs it, and works for any future
 * notification type without per-type frontend code.
 */
export function renderNotificationText(
	template: string,
	params: Record<string, string>,
): string {
	return template.replace(
		/\{\{(\w+)\}\}/g,
		(_, key: string) => params[key] ?? "",
	);
}

/**
 * Renders a "3 minutes ago" style string, tolerating a missing/malformed
 * date instead of throwing — a breaking API field-shape change once left
 * stale, badly-shaped notifications sitting in a user's sessionStorage,
 * and date-fns throws RangeError on an invalid Date rather than returning
 * a fallback.
 */
export function formatRelativeTime(
	dateValue: string | undefined | null,
	lang: string | undefined,
): string {
	if (!dateValue) return "";
	const date = new Date(dateValue);
	if (Number.isNaN(date.getTime())) return "";
	return formatDistanceToNow(date, {
		addSuffix: true,
		locale: lang === "en" ? enUS : es,
	});
}
