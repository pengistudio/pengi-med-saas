/**
 * WhatsApp's customer service window: free-text replies are allowed only up to
 * 24 h after the patient's last message. The backend sends when it closes
 * (`window_expires_at`, null if the patient never wrote) and rejects replies
 * after it; this evaluates it against the browser clock so the composer locks
 * and the countdown moves between polls.
 */
export interface ReplyWindow {
	open: boolean;
	/** Whole hours left (0 in the last hour); 0 when closed. */
	hoursLeft: number;
	/** Whole minutes left, at least 1 while open; 0 when closed. */
	minutesLeft: number;
}

const MINUTE_MS = 60_000;
const HOUR_MS = 60 * MINUTE_MS;

export function replyWindow(
	expiresAt: string | null | undefined,
	now: Date = new Date(),
): ReplyWindow {
	const closed: ReplyWindow = { open: false, hoursLeft: 0, minutesLeft: 0 };
	if (!expiresAt) return closed;
	const end = new Date(expiresAt).getTime();
	if (Number.isNaN(end)) return closed;
	const left = end - now.getTime();
	if (left <= 0) return closed;
	return {
		open: true,
		hoursLeft: Math.floor(left / HOUR_MS),
		minutesLeft: Math.max(1, Math.floor(left / MINUTE_MS)),
	};
}
