import type { Notification } from "@/api/notification-service";

export const WHATSAPP_MESSAGE_NOTIFICATION = "whatsapp.message";

/** Notification id → its UpdatedAt when last seen. */
export type SeenNotifications = Map<number, string>;

/** Ids kept in memory; older ones drop out first. */
const SEEN_LIMIT = 200;

/**
 * Compares a poll's notifications with the ones already seen and returns the
 * WhatsApp messages that are new: an unseen id, or a seen one whose UpdatedAt
 * moved (the backend updates the conversation's unread notification instead
 * of creating another). `seen` null means the first poll: nothing is new, it
 * only records what is there.
 */
export function detectNewWhatsAppMessages(
	seen: SeenNotifications | null,
	items: readonly Notification[],
): { fresh: Notification[]; seen: SeenNotifications } {
	const next: SeenNotifications = new Map(seen ?? []);
	const fresh: Notification[] = [];
	for (const n of items) {
		const stamp = n.UpdatedAt ?? n.CreatedAt ?? "";
		if (
			seen &&
			n.type === WHATSAPP_MESSAGE_NOTIFICATION &&
			seen.get(n.ID) !== stamp
		) {
			fresh.push(n);
		}
		next.delete(n.ID);
		next.set(n.ID, stamp);
	}
	while (next.size > SEEN_LIMIT) {
		const oldest = next.keys().next().value;
		if (oldest === undefined) break;
		next.delete(oldest);
	}
	return { fresh, seen: next };
}

/** The inbox already shows new messages: no toast there. */
export function isOnWhatsAppInbox(pathname: string): boolean {
	return pathname === "/whatsapp" || pathname.startsWith("/whatsapp/");
}

/**
 * Notification params are JSON (strings or numbers); `count` must be a number for textGet to
 * pick the plural form (`key.one` / `key.other`).
 */
export function notificationTextParams(
	params: Record<string, unknown> | null | undefined,
): Record<string, string | number> {
	const out: Record<string, string | number> = {};
	for (const [k, v] of Object.entries(params ?? {})) {
		if (typeof v === "string" || typeof v === "number") out[k] = v;
		else if (v !== null && v !== undefined) out[k] = String(v);
	}
	const count = Number(out.count);
	if (out.count !== undefined && out.count !== "" && Number.isFinite(count)) {
		out.count = count;
	}
	return out;
}
