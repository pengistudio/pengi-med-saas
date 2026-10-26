import { toDateOnlyString, useText } from "@pengi/shared";
import { cn } from "@pengi/ui";
import { CalendarClock } from "lucide-react";
import { getAppointments, type PortalAppointment } from "@/api/portal-service";
import { byYear, Ledger, LedgerRow, LedgerYear } from "@/components/ledger";
import { SectionState } from "@/components/section-state";
import { nameCase } from "@/lib/person-name";
import { useLoad } from "@/lib/use-load";

const UPCOMING_STATUS = ["confirmed", "cancelled"];
// Once the day has passed, the front-desk steps (scheduled, arrived, …) mean
// nothing to the patient: only "attended" or "cancelled" is worth saying.
const PAST_STATUS = ["completed", "cancelled"];

const hhmm = (time: string) => time.slice(0, 5);
const byStart = (a: PortalAppointment, b: PortalAppointment) =>
	`${a.date}${a.start_time}`.localeCompare(`${b.date}${b.start_time}`);

function useSpecialty() {
	const { textGet } = useText();
	return (a: PortalAppointment) =>
		a.specialty === "other"
			? a.specialty_other
			: a.specialty && textGet(`doctors.specialty.${a.specialty}`);
}

function StatusText({ status, shown }: { status: string; shown: string[] }) {
	const { textGet } = useText();
	if (!shown.includes(status)) return null;
	return (
		<p
			className={cn(
				"text-sm",
				status === "cancelled" ? "text-destructive" : "text-primary",
			)}
		>
			{textGet(`appointments.status.${status}`)}
		</p>
	);
}

/** The one thing a patient opens the portal for: when and with whom. */
function NextAppointment({ appointment }: { appointment: PortalAppointment }) {
	const { textGet, formatDate } = useText();
	const specialty = useSpecialty()(appointment);

	return (
		<section
			aria-labelledby="next-appointment"
			className="rounded-3xl bg-secondary px-5 py-6 text-secondary-foreground"
		>
			<h2 id="next-appointment" className="text-sm font-medium">
				{textGet("portal.appointments.next")}
			</h2>
			<p className="mt-3 text-3xl font-semibold tracking-tight text-foreground first-letter:uppercase">
				{formatDate(appointment.date, "weekday-day-month")}
			</p>
			<p className="text-3xl font-semibold tracking-tight text-primary tabular-nums">
				{hhmm(appointment.start_time)}
			</p>
			<div className="mt-5 space-y-0.5 text-foreground">
				{appointment.doctor_name && (
					<p className="font-medium">{nameCase(appointment.doctor_name)}</p>
				)}
				{specialty && <p className="text-muted-foreground">{specialty}</p>}
				{appointment.location && (
					<p className="text-muted-foreground">{appointment.location}</p>
				)}
				<StatusText status={appointment.status} shown={UPCOMING_STATUS} />
			</div>
		</section>
	);
}

function AppointmentRow({
	appointment,
	past,
}: {
	appointment: PortalAppointment;
	past: boolean;
}) {
	const specialty = useSpecialty()(appointment);
	return (
		<LedgerRow
			date={appointment.date}
			muted={past || appointment.status === "cancelled"}
		>
			<p className="font-medium tabular-nums">
				{hhmm(appointment.start_time)}
				{appointment.doctor_name && (
					<span className="font-normal">
						{" "}
						{nameCase(appointment.doctor_name)}
					</span>
				)}
			</p>
			{specialty && (
				<p className="text-sm text-muted-foreground">{specialty}</p>
			)}
			<StatusText
				status={appointment.status}
				shown={past ? PAST_STATUS : UPCOMING_STATUS}
			/>
		</LedgerRow>
	);
}

export function AppointmentsSection() {
	const { textGet } = useText();
	const state = useLoad(getAppointments);

	return (
		<SectionState
			state={state}
			emptyKey="portal.appointments.empty"
			isEmpty={(list) => list.length === 0}
		>
			{(list) => {
				const today = toDateOnlyString(new Date());
				const upcoming = list.filter((a) => a.date >= today).sort(byStart);
				const past = list
					.filter((a) => a.date < today)
					.sort((a, b) => byStart(b, a));
				const [next, ...later] = upcoming.filter(
					(a) => a.status !== "cancelled",
				);
				const thisYear = today.slice(0, 4);

				return (
					<div className="space-y-8">
						{next ? (
							<NextAppointment appointment={next} />
						) : (
							<div className="flex items-start gap-3 rounded-3xl border border-dashed px-5 py-5">
								<CalendarClock className="mt-0.5 size-5 shrink-0 text-primary" />
								<p className="text-sm text-muted-foreground">
									{textGet("portal.appointments.none_upcoming")}
								</p>
							</div>
						)}

						{later.length > 0 && (
							<section>
								<h2 className="text-base font-semibold">
									{textGet("portal.appointments.later")}
								</h2>
								<Ledger>
									{later.map((a) => (
										<AppointmentRow key={a.id} appointment={a} past={false} />
									))}
								</Ledger>
							</section>
						)}

						{past.length > 0 && (
							<section>
								<h2 className="text-base font-semibold">
									{textGet("portal.appointments.past")}
								</h2>
								{byYear(past).map(([year, items]) => (
									<div key={year}>
										{year !== thisYear && <LedgerYear year={year} />}
										<Ledger>
											{items.map((a) => (
												<AppointmentRow key={a.id} appointment={a} past />
											))}
										</Ledger>
									</div>
								))}
							</section>
						)}
					</div>
				);
			}}
		</SectionState>
	);
}
