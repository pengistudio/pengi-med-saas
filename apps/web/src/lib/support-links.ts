import type { AppText } from "@pengi/shared";
import { SUPPORT } from "@/lib/constants";
import { generateWhatsAppLink } from "@/lib/utils";

/** Where the user asks for help from: their clinic and the page they are on. */
export interface SupportContext {
	clinic: string;
	slug: string;
	page: string;
}

/**
 * Links of the help menu, with the clinic and the page already written in
 * the message so support knows the context without asking.
 */
export function supportLinks(
	context: SupportContext,
	textGet: AppText["textGet"],
) {
	const params = { ...context };
	const subject = textGet("dashboard.help.email.subject", params);
	const body = textGet("dashboard.help.email.body", params);
	return {
		whatsapp: generateWhatsAppLink(
			SUPPORT.WHATSAPP,
			textGet("dashboard.help.whatsapp.message", params),
		),
		email: `mailto:${SUPPORT.EMAIL}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`,
		report: generateWhatsAppLink(
			SUPPORT.WHATSAPP,
			textGet("dashboard.help.report.message", params),
		),
	};
}
