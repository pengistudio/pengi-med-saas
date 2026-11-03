import type {
	WhatsAppBotFAQEntry,
	WhatsAppBotReadiness,
} from "@/api/whatsapp-service";

/**
 * The readiness checklist of the WhatsApp AI assistant, in the order the
 * settings card shows it. `to` is where the admin fixes a missing item.
 */
export const BOT_READINESS_ITEMS: {
	key: Exclude<keyof WhatsAppBotReadiness, "plan_whatsapp" | "plan_ai">;
	to?: string;
}[] = [
	{ key: "whatsapp_connected" },
	{ key: "ai_opt_in" },
	{ key: "provider_configured" },
	{ key: "office_hours", to: "/settings/agenda" },
	{ key: "address", to: "/billing/settings" },
	{ key: "faq" },
];

/** A pair the server keeps: both sides written. */
export function faqEntryComplete(entry: WhatsAppBotFAQEntry): boolean {
	return entry.question.trim() !== "" && entry.answer.trim() !== "";
}

/** The FAQ as it will be saved: trimmed, fully empty pairs dropped. */
export function cleanFAQ(
	entries: WhatsAppBotFAQEntry[],
): WhatsAppBotFAQEntry[] {
	return entries
		.map((e) => ({ question: e.question.trim(), answer: e.answer.trim() }))
		.filter((e) => e.question !== "" || e.answer !== "");
}

/**
 * Whether the FAQ draft can be saved: every kept pair complete and inside
 * the limits (the server checks the same).
 */
export function faqDraftValid(
	entries: WhatsAppBotFAQEntry[],
	limits: { max_entries: number; max_question: number; max_answer: number },
): boolean {
	const kept = cleanFAQ(entries);
	return (
		kept.length <= limits.max_entries &&
		kept.every(
			(e) =>
				faqEntryComplete(e) &&
				e.question.length <= limits.max_question &&
				e.answer.length <= limits.max_answer,
		)
	);
}

/** The readiness with the FAQ item judged on the draft being edited. */
export function readinessWithDraft(
	readiness: WhatsAppBotReadiness,
	draft: WhatsAppBotFAQEntry[],
): WhatsAppBotReadiness {
	return { ...readiness, faq: cleanFAQ(draft).some(faqEntryComplete) };
}

/** Every requirement met: the switch can be turned on. */
export function botReady(readiness: WhatsAppBotReadiness): boolean {
	return Object.values(readiness).every(Boolean);
}
