import { parseDateOnly, useText } from "@pengi/shared";
import { Badge } from "@pengi/ui";
import { Link2, UserPlus } from "lucide-react";
import type { AppointmentRequest } from "@/api/appointment-request-service";
import { FormattedDate } from "@/components/custom/formatted";
import { ageInYears } from "@/lib/patient-age";

/**
 * Who asked for the appointment. A new patient (no record yet) shows what
 * they typed, how they verified, and whether approving links to an existing
 * record or creates one.
 */
export function RequesterCell({ request }: { request: AppointmentRequest }) {
	const { textGet } = useText();
	const { patient, new_patient: person, matching_patient: match } = request;

	if (!person) {
		if (!patient) return <span className="text-muted-foreground">—</span>;
		return (
			<div className="min-w-0">
				<p className="font-medium">
					{patient.first_name} {patient.last_name}
				</p>
				<p className="text-sm text-muted-foreground">
					{[patient.document, patient.phone].filter(Boolean).join(" · ")}
				</p>
			</div>
		);
	}

	const age = ageInYears(parseDateOnly(person.birth_date));
	const pending = request.status === "pending";

	return (
		<div className="min-w-0 space-y-1">
			<div className="flex flex-wrap items-center gap-2">
				<p className="font-medium">
					{person.first_name} {person.last_name}
				</p>
				<Badge variant="secondary" className="gap-1">
					<UserPlus className="h-3 w-3" />
					{textGet("appointment_requests.new_patient.badge")}
				</Badge>
			</div>
			<p className="text-sm text-muted-foreground">
				{textGet(
					person.document_type === "passport"
						? "appointment_requests.new_patient.passport"
						: "appointment_requests.new_patient.cedula",
				)}{" "}
				<span className="tabular-nums">{person.document}</span>
				{person.birth_date && (
					<>
						{" · "}
						<FormattedDate value={person.birth_date} style="medium" />
						{age !== null &&
							` (${textGet("clinical.patient_card.age_years", { count: age })})`}
					</>
				)}
			</p>
			<p className="text-sm text-muted-foreground break-all">
				{[person.phone, person.email].filter(Boolean).join(" · ")}
			</p>
			<p className="text-xs text-muted-foreground">
				{textGet(
					person.contact_channel === "whatsapp"
						? "appointment_requests.new_patient.verified.whatsapp"
						: "appointment_requests.new_patient.verified.email",
				)}
			</p>
			{pending && (
				<p className="flex items-start gap-1.5 text-xs">
					<Link2 className="mt-0.5 h-3 w-3 shrink-0" />
					{match
						? textGet("appointment_requests.new_patient.matching", {
								name: `${match.first_name} ${match.last_name}`,
							})
						: textGet("appointment_requests.new_patient.will_create")}
				</p>
			)}
		</div>
	);
}
