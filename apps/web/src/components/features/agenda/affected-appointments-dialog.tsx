import { useText } from "@pengi/shared";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";
import type { AffectedAppointment } from "@/api/agenda-service";
import { STATUS_I18N_KEYS } from "@/components/features/appointments/appointment-utils";
import { findDoctor, useDoctors } from "@/store/doctors-store";

/**
 * After saving a block: the appointments it covers. Nothing was cancelled;
 * the list is there so the clinic can reschedule or notify them.
 */
export function AffectedAppointmentsDialog({
	appointments,
	onClose,
}: {
	appointments: AffectedAppointment[] | null;
	onClose: () => void;
}) {
	const { textGet, formatDate } = useText();
	const { doctors } = useDoctors(!!appointments);
	const open = !!appointments && appointments.length > 0;

	return (
		<Dialog
			open={open}
			onOpenChange={(next) => {
				if (!next) onClose();
			}}
		>
			<DialogContent className="sm:max-w-lg">
				<DialogHeader>
					<DialogTitle>{textGet("agenda.blocks.affected.title")}</DialogTitle>
					<DialogDescription>
						{textGet("agenda.blocks.affected.description", {
							count: appointments?.length ?? 0,
						})}
					</DialogDescription>
				</DialogHeader>
				<ul className="max-h-[50vh] divide-y overflow-y-auto rounded-lg border">
					{appointments?.map((a) => {
						const doctor = findDoctor(doctors, a.doctor_id);
						return (
							<li key={a.id} className="grid gap-0.5 px-3 py-2 text-sm">
								<span className="font-medium">{a.patient_name}</span>
								<span className="text-muted-foreground">
									{formatDate(a.date, "weekday-day-month")} · {a.start_time}–
									{a.end_time}
									{doctor ? ` · ${doctor.full_name}` : ""}
									{" · "}
									{textGet(
										STATUS_I18N_KEYS[a.status] ??
											"appointments.status.scheduled",
									)}
								</span>
							</li>
						);
					})}
				</ul>
				<DialogFooter>
					<Button onClick={onClose}>
						{textGet("agenda.blocks.affected.ok")}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
