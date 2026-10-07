import type { WhatsAppTemplate, WhatsAppUsage } from "@/api/whatsapp-service";

/** The share of the cap at which the usage shows as a warning (the backend warns at 80% too). */
export const USAGE_WARNING_RATIO = 0.8;

export interface UsageSummary {
	unlimited: boolean;
	/** 0–100, rounded; 0 when unlimited. */
	percent: number;
	/** At or above the warning share, but below the cap. */
	warning: boolean;
	/** The cap is reached: templates are blocked until next month. */
	reached: boolean;
}

/** How full the month's template quota is. A limit of -1 means unlimited. */
export function usageSummary(
	usage: Pick<WhatsAppUsage, "used" | "limit">,
): UsageSummary {
	if (usage.limit < 0) {
		return { unlimited: true, percent: 0, warning: false, reached: false };
	}
	if (usage.limit === 0) {
		return { unlimited: false, percent: 100, warning: false, reached: true };
	}
	const ratio = usage.used / usage.limit;
	const reached = usage.used >= usage.limit;
	return {
		unlimited: false,
		percent: Math.min(100, Math.round(ratio * 100)),
		warning: !reached && ratio >= USAGE_WARNING_RATIO,
		reached,
	};
}

/** The template needs one of the patient's appointments to fill its date and time. */
export function templateNeedsAppointment(
	t: Pick<WhatsAppTemplate, "needs_appointment" | "variables">,
): boolean {
	return (
		t.needs_appointment ||
		t.variables.includes("date") ||
		t.variables.includes("time")
	);
}

/** The text to preview: the body filled with sample values, else the raw body. */
export function templatePreview(
	t: Pick<WhatsAppTemplate, "preview" | "body">,
): string {
	return t.preview || t.body;
}

/** Templates that can be sent from the inbox, usable ones first. */
export function inboxTemplates(
	templates: readonly WhatsAppTemplate[],
): WhatsAppTemplate[] {
	return templates
		.filter((t) => t.inbox)
		.sort((a, b) => Number(b.usable) - Number(a.usable));
}
