import { parseDateOnly, useText } from "@pengi/shared";
import { Badge } from "@pengi/ui";
import { AlertTriangle, UserX } from "lucide-react";
import React from "react";
import { getPatientNoShows, type PatientNoShows } from "@/api/clinical-service";

/** No-shows in the last 12 months from which booking shows a warning. */
export const NO_SHOW_WARNING_THRESHOLD = 2;

/**
 * The patient's no-show history (null while loading, for no patient, or when
 * the request fails: it is only a hint).
 */
export function usePatientNoShows(patientId: number | null | undefined) {
	const [answer, setAnswer] = React.useState<{
		patientId: number;
		data: PatientNoShows;
	} | null>(null);

	React.useEffect(() => {
		if (!patientId) return;
		let current = true;
		getPatientNoShows(patientId).then((res) => {
			if (current && res.success && res.data) {
				setAnswer({ patientId, data: res.data });
			}
		});
		return () => {
			current = false;
		};
	}, [patientId]);

	// Tagged with the patient it answers, so another patient's count never shows.
	return patientId && answer?.patientId === patientId ? answer.data : null;
}

/**
 * A non-blocking notice when booking for a patient who missed
 * NO_SHOW_WARNING_THRESHOLD or more appointments in the last 12 months.
 */
export function NoShowWarning({ patientId }: { patientId: number | null }) {
	const { textGet } = useText();
	const noShows = usePatientNoShows(patientId);
	if (!noShows || noShows.count < NO_SHOW_WARNING_THRESHOLD) return null;
	return (
		<p
			role="status"
			className="mt-2 flex items-start gap-2 rounded-lg border border-rose-500/40 bg-rose-500/10 px-3 py-2 text-sm text-rose-800 dark:text-rose-300"
		>
			<AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
			{textGet("appointments.no_shows.warning", { count: noShows.count })}
		</p>
	);
}

/** The patient file's badge: no-shows in the last 12 months and the latest. */
export function NoShowBadge({ patientId }: { patientId: number }) {
	const { textGet, formatDate } = useText();
	const noShows = usePatientNoShows(patientId);
	if (!noShows || noShows.count === 0) return null;
	const last = parseDateOnly(noShows.last_date);
	return (
		<Badge
			variant="outline"
			className="border-rose-300 bg-rose-50 font-medium text-rose-800 dark:border-rose-800 dark:bg-rose-950 dark:text-rose-300"
		>
			<UserX />
			{textGet("appointments.no_shows.badge", { count: noShows.count })}
			{last && (
				<span className="font-normal">
					{" · "}
					{textGet("appointments.no_shows.last", {
						date: formatDate(last, "long"),
					})}
				</span>
			)}
		</Badge>
	);
}
