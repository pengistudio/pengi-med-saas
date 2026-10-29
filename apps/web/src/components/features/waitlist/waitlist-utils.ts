import type { AppText } from "@pengi/shared";
import type { WaitlistEntry, WaitlistStatus } from "@/api/waitlist-service";

export const WAITLIST_STATUS_VARIANT: Record<
	WaitlistStatus,
	"default" | "secondary" | "outline" | "destructive"
> = {
	waiting: "default",
	offered: "secondary",
	booked: "outline",
	removed: "outline",
	expired: "outline",
};

/**
 * What the patient waits for, in one line: the type, the preferred range and
 * the appointment to move earlier ("Consulta · Del 3 nov. al 10 nov.").
 */
export function entryPreference(
	entry: Pick<
		WaitlistEntry,
		"appointment_type" | "date_from" | "date_to" | "appointment"
	>,
	textGet: AppText["textGet"],
	formatDate: AppText["formatDate"],
): string {
	const parts: string[] = [
		entry.appointment_type?.name ?? textGet("waitlist.type.any"),
	];
	const from = entry.date_from && formatDate(entry.date_from, "day-month");
	const to = entry.date_to && formatDate(entry.date_to, "day-month");
	if (from && to) parts.push(textGet("waitlist.range.between", { from, to }));
	else if (from) parts.push(textGet("waitlist.range.from", { from }));
	else if (to) parts.push(textGet("waitlist.range.until", { to }));
	if (entry.appointment?.date) {
		parts.push(
			textGet("waitlist.move_earlier", {
				date: formatDate(entry.appointment.date, "day-month"),
				time: entry.appointment.start_time,
			}),
		);
	}
	return parts.join(" · ");
}
