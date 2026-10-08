import { toDateOnlyString, useText } from "@pengi/shared";
import { AlertTriangle } from "lucide-react";
import React from "react";
import { type Control, useWatch } from "react-hook-form";
import { type Availability, getAvailability } from "@/api/agenda-service";
import {
	type AppointmentFormValues,
	timeToMinutes,
} from "@/components/features/appointments/appointment-utils";

const TIME = /^\d{2}:\d{2}$/;
const DEBOUNCE_MS = 400;

/**
 * A non-blocking notice under the appointment times when the slot falls
 * outside the doctor's working hours or inside a block. Nothing when it is
 * inside, or when the doctor has no schedule.
 */
export function AvailabilityWarning({
	control,
}: {
	control: Control<AppointmentFormValues>;
}) {
	const { textGet } = useText();
	const doctorId = useWatch({ control, name: "doctor_id" });
	const date = useWatch({ control, name: "date" });
	const start = useWatch({ control, name: "start_time" });
	const end = useWatch({ control, name: "end_time" });
	// Tagged with the slot it answers, so an answer for a previous slot is
	// never shown while the new one is pending.
	const [answer, setAnswer] = React.useState<{
		key: string;
		data: Availability;
	} | null>(null);

	const day = date instanceof Date ? toDateOnlyString(date) : "";
	const valid =
		!!doctorId &&
		!!day &&
		TIME.test(start ?? "") &&
		TIME.test(end ?? "") &&
		timeToMinutes(start) < timeToMinutes(end);
	const key = `${doctorId}|${day}|${start}|${end}`;

	React.useEffect(() => {
		if (!valid || !doctorId) return;
		let current = true;
		const timer = setTimeout(() => {
			getAvailability({
				doctor_id: doctorId,
				date: day,
				start_time: start,
				end_time: end,
			}).then((res) => {
				if (current && res.success && res.data) {
					setAnswer({ key, data: res.data });
				}
			});
		}, DEBOUNCE_MS);
		return () => {
			current = false;
			clearTimeout(timer);
		};
	}, [valid, key, doctorId, day, start, end]);

	const result = answer?.key === key ? answer.data : null;
	if (!valid || !result) return null;
	let message: string | null = null;
	if (result.status === "blocked") {
		const reasons = result.blocks
			.map((b) => b.reason)
			.filter(Boolean)
			.join(", ");
		message = textGet("agenda.availability.blocked", {
			reason: reasons || textGet("agenda.blocks.no_reason"),
		});
	} else if (result.status === "outside") {
		message =
			result.ranges.length === 0
				? textGet("agenda.availability.day_off")
				: textGet("agenda.availability.outside", {
						ranges: result.ranges
							.map((r) => `${r.start_time}–${r.end_time}`)
							.join(", "),
					});
	}
	if (!message) return null;

	return (
		<p
			role="status"
			className="flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-800 dark:text-amber-300"
		>
			<AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
			{message}
		</p>
	);
}
