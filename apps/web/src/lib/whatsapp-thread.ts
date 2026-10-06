/** What a conversation thread needs of a message to merge pages. */
interface ThreadItem {
	id: number;
}

/**
 * Merges a fetched page (the newest page from a poll, an older page, or a just
 * sent reply) into the loaded thread: fresh copies replace stale ones (status
 * changes), and the result stays oldest first by id.
 */
export function mergeMessages<T extends ThreadItem>(
	loaded: readonly T[],
	incoming: readonly T[],
): T[] {
	const byId = new Map<number, T>();
	for (const m of loaded) byId.set(m.id, m);
	for (const m of incoming) byId.set(m.id, m);
	return [...byId.values()].sort((a, b) => a.id - b.id);
}

/** WhatsApp numbers are stored as digits with the country code: "+593…". */
export function displayPhone(phone: string): string {
	return /^\d+$/.test(phone) ? `+${phone}` : phone;
}

/** The linked patient's name, or the number when it isn't linked to one. */
export function conversationTitle(c: {
	patient_name: string;
	phone: string;
}): string {
	return c.patient_name || displayPhone(c.phone);
}
