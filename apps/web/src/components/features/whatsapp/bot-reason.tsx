import { Text } from "@pengi/ui";
import type { ReactNode } from "react";

/**
 * Why the WhatsApp AI assistant handed a chat to reception (or stayed
 * silent): the server's reason codes, each with its literal message key so
 * the catalog test checks them.
 */
const REASONS: Record<string, ReactNode> = {
	asked_person: <Text uuid="whatsapp.bot.reason.asked_person" />,
	emergency: <Text uuid="whatsapp.bot.reason.emergency" />,
	symptoms: <Text uuid="whatsapp.bot.reason.symptoms" />,
	non_text: <Text uuid="whatsapp.bot.reason.non_text" />,
	too_long: <Text uuid="whatsapp.bot.reason.too_long" />,
	suspicious_text: <Text uuid="whatsapp.bot.reason.suspicious_text" />,
	not_in_faq: <Text uuid="whatsapp.bot.reason.not_in_faq" />,
	guard: <Text uuid="whatsapp.bot.reason.guard" />,
	unparseable: <Text uuid="whatsapp.bot.reason.unparseable" />,
	quota: <Text uuid="whatsapp.bot.reason.quota" />,
	not_configured: <Text uuid="whatsapp.bot.reason.not_configured" />,
	provider_error: <Text uuid="whatsapp.bot.reason.provider_error" />,
	daily_limit: <Text uuid="whatsapp.bot.reason.daily_limit" />,
	clinic_daily_limit: <Text uuid="whatsapp.bot.reason.clinic_daily_limit" />,
	overloaded: <Text uuid="whatsapp.bot.reason.overloaded" />,
	send_failed: <Text uuid="whatsapp.bot.reason.send_failed" />,
	human_active: <Text uuid="whatsapp.bot.reason.human_active" />,
	awaiting_human: <Text uuid="whatsapp.bot.reason.awaiting_human" />,
	opted_out: <Text uuid="whatsapp.bot.reason.opted_out" />,
	outside_window: <Text uuid="whatsapp.bot.reason.outside_window" />,
	consent_answer: <Text uuid="whatsapp.bot.reason.consent_answer" />,
	internal: <Text uuid="whatsapp.bot.reason.internal" />,
};

/** The reason's text; nothing for an unknown code. */
export function BotReason({ reason }: { reason: string }) {
	return <>{REASONS[reason] ?? null}</>;
}
