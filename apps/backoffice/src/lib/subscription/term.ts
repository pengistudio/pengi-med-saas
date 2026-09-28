import type { PricingOption } from "@/api/plan-service";

/**
 * Subscriptions are sold on Ecuador's calendar: "expires on the 16th" means
 * until the end of the 16th there. The API receives the date (YYYY-MM-DD) and
 * stores that end of day; the backoffice shows and edits the Ecuador date.
 */
export const SUBSCRIPTION_TIME_ZONE = "America/Guayaquil";

// en-CA formats dates as YYYY-MM-DD.
const isoDateIn = (zone: string) =>
	new Intl.DateTimeFormat("en-CA", {
		timeZone: zone,
		year: "numeric",
		month: "2-digit",
		day: "2-digit",
	});
const ecuadorDate = isoDateIn(SUBSCRIPTION_TIME_ZONE);

/** Today's date (YYYY-MM-DD) in Ecuador. */
export function todayInEcuador(now = new Date()): string {
	return ecuadorDate.format(now);
}

/**
 * date + months, keeping the day when it exists in the target month and using
 * its last day otherwise (31 Jan + 1 month = 28/29 Feb). Same rule as the API.
 */
export function addMonths(date: string, months: number): string {
	const [year, month, day] = date.split("-").map(Number);
	const target = new Date(Date.UTC(year, month - 1 + months, 1));
	const lastDay = new Date(
		Date.UTC(target.getUTCFullYear(), target.getUTCMonth() + 1, 0),
	).getUTCDate();
	target.setUTCDate(Math.min(day, lastDay));
	return target.toISOString().slice(0, 10);
}

/** The expiry date to suggest for a period of `months` starting today. */
export function suggestExpiry(months: number, now = new Date()): string {
	return addMonths(todayInEcuador(now), months);
}

/** The Ecuador date (YYYY-MM-DD) of an expiry timestamp from the API. */
export function expiryDate(expiresAt: string): string {
	return ecuadorDate.format(new Date(expiresAt));
}

export function sortedPricings(
	plan: { pricings?: PricingOption[] } | undefined,
): PricingOption[] {
	return [...(plan?.pricings ?? [])].sort((a, b) => a.months - b.months);
}
