import { toDateOnlyString, useText } from "@pengi/shared";
import { Button, buttonVariants, cn } from "@pengi/ui";
import { CalendarClock, CalendarPlus, Hourglass } from "lucide-react";
import React from "react";
import { Link } from "react-router";
import {
	cancelAppointment,
	cancelBookingRequest,
	getAppointments,
	getBookingRequests,
	type Me,
	type PortalAppointment,
	type PortalAppointmentRequest,
} from "@/api/portal-service";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { byYear, Ledger, LedgerRow, LedgerYear } from "@/components/ledger";
import { SectionState } from "@/components/section-state";
import { nameCase } from "@/lib/person-name";
import { useSpecialtyLabel } from "@/lib/specialty";
import { useReloadableLoad } from "@/lib/use-load";

const UPCOMING_STATUS = ["confirmed", "cancelled"];
// Once the day has passed, the front-desk steps (scheduled, arrived, …) mean
// nothing to the patient: only "attended" or "cancelled" is worth saying.
const PAST_STATUS = ["completed", "cancelled"];
// Statuses the patient may still cancel (the API's `can_cancel` decides the time).
const CANCELLABLE_STATUS = ["scheduled", "confirmed"];
// Requests that did not become an appointment; the patient's own cancels are left out.
const CLOSED_REQUEST_STATUS = ["rejected", "expired"];

const hhmm = (time: string) => time.slice(0, 5);
const byStart = (
	a: { date: string; start_time: string },
	b: { date: string; start_time: string },
) => `${a.date}${a.start_time}`.localeCompare(`${b.date}${b.start_time}`);

/** What the patient may cancel; set by a button, read by the one dialog. */
type CancelTarget =
	| { kind: "appointment"; appointment: PortalAppointment }
	| { kind: "request"; request: PortalAppointmentRequest };

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

/** "Cancel" when the clinic's notice allows it, "call the clinic" when not. */
function CancelControl({
	appointment,
	booking,
	onCancel,
	className,
	onTint = false,
}: {
	appointment: PortalAppointment;
	booking: boolean;
	onCancel: (appointment: PortalAppointment) => void;
	className?: string;
	/** On the tinted card: the button needs a white face to stand out. */
	onTint?: boolean;
}) {
	const { textGet } = useText();
	if (!CANCELLABLE_STATUS.includes(appointment.status)) return null;
	if (appointment.can_cancel) {
		return (
			<Button
				type="button"
				variant="secondary"
				size="sm"
				className={cn(
					"rounded-full",
					onTint && "bg-card hover:bg-card/80",
					className,
				)}
				onClick={() => onCancel(appointment)}
			>
				{textGet("portal.booking.cancel_appointment")}
			</Button>
		);
	}
	if (!booking) return null;
	return (
		<p className={cn("text-sm text-muted-foreground", className)}>
			{textGet("portal.booking.call_to_cancel")}
		</p>
	);
}

/**
 * The one thing a patient opens the portal for: when and with whom. From
 * `sm` the day is set large on its own, like the page of an appointment book.
 */
function NextAppointment({
	appointment,
	booking,
	onCancel,
}: {
	appointment: PortalAppointment;
	booking: boolean;
	onCancel: (appointment: PortalAppointment) => void;
}) {
	const { textGet, formatDate } = useText();
	const specialty = useSpecialtyLabel()(appointment);

	return (
		<section
			aria-labelledby="next-appointment"
			className="rounded-3xl bg-secondary px-5 py-6 text-secondary-foreground sm:flex sm:gap-8 sm:px-8 sm:py-8"
		>
			<div
				aria-hidden="true"
				className="hidden w-28 shrink-0 flex-col items-center justify-center border-primary/15 border-r pr-8 leading-none tabular-nums sm:flex"
			>
				<span className="text-sm text-muted-foreground first-letter:uppercase">
					{formatDate(appointment.date, "weekday")}
				</span>
				<span className="mt-2 text-7xl font-semibold tracking-tighter text-foreground">
					{Number(appointment.date.slice(8, 10))}
				</span>
				<span className="mt-2 text-sm text-muted-foreground">
					{formatDate(appointment.date, "month")}
				</span>
			</div>
			<div className="min-w-0 flex-1">
				<h2 id="next-appointment" className="text-sm font-medium">
					{textGet("portal.appointments.next")}
				</h2>
				<p className="mt-3 text-3xl font-semibold tracking-tight text-foreground first-letter:uppercase sm:sr-only">
					{formatDate(appointment.date, "weekday-day-month")}
				</p>
				<p className="text-3xl font-semibold tracking-tight text-primary tabular-nums sm:mt-3 sm:text-5xl">
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
				<CancelControl
					appointment={appointment}
					booking={booking}
					onCancel={onCancel}
					onTint
					className="mt-5"
				/>
			</div>
		</section>
	);
}

function AppointmentRow({
	appointment,
	past,
	booking,
	onCancel,
}: {
	appointment: PortalAppointment;
	past: boolean;
	booking: boolean;
	onCancel: (appointment: PortalAppointment) => void;
}) {
	const specialty = useSpecialtyLabel()(appointment);
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
			{!past && (
				<CancelControl
					appointment={appointment}
					booking={booking}
					onCancel={onCancel}
					className="mt-2"
				/>
			)}
		</LedgerRow>
	);
}

/** A request the clinic has not answered yet: held time, and until when. */
function PendingRequestRow({
	request,
	onCancel,
}: {
	request: PortalAppointmentRequest;
	onCancel: (request: PortalAppointmentRequest) => void;
}) {
	const { textGet, formatRelative } = useText();
	const specialty = useSpecialtyLabel()(request.doctor);
	return (
		<LedgerRow date={request.date}>
			<p className="font-medium tabular-nums">
				{hhmm(request.start_time)}
				{request.doctor.full_name && (
					<span className="font-normal">
						{" "}
						{nameCase(request.doctor.full_name)}
					</span>
				)}
			</p>
			<p className="text-sm text-muted-foreground">
				{[request.type_name, specialty].filter(Boolean).join(" · ")}
			</p>
			<p className="flex items-center gap-1.5 text-sm text-primary">
				<Hourglass aria-hidden="true" className="size-3.5 shrink-0" />
				{textGet("portal.booking.pending")}
			</p>
			{request.expires_at && (
				<p className="text-xs text-muted-foreground">
					{textGet("portal.booking.expires", {
						when: formatRelative(request.expires_at),
					})}
				</p>
			)}
			<Button
				type="button"
				variant="secondary"
				size="sm"
				className="mt-2 rounded-full"
				onClick={() => onCancel(request)}
			>
				{textGet("portal.booking.cancel_request")}
			</Button>
		</LedgerRow>
	);
}

/** A request that lapsed or the clinic turned down, with its reason. */
function ClosedRequestRow({ request }: { request: PortalAppointmentRequest }) {
	const { textGet } = useText();
	return (
		<LedgerRow date={request.date} muted>
			<p className="font-medium tabular-nums text-muted-foreground">
				{hhmm(request.start_time)}
				{request.doctor.full_name && (
					<span className="font-normal">
						{" "}
						{nameCase(request.doctor.full_name)}
					</span>
				)}
			</p>
			<p className="text-sm text-muted-foreground">
				{textGet(`portal.booking.status.${request.status}`)}
			</p>
			{request.reject_reason && (
				<p className="text-sm text-muted-foreground">
					{textGet("portal.booking.reject_reason", {
						reason: request.reject_reason,
					})}
				</p>
			)}
		</LedgerRow>
	);
}

/** The patient's booking requests; only shown when the clinic takes them. */
function useRequests(booking: boolean) {
	const [state, reload] = useReloadableLoad(() =>
		booking
			? getBookingRequests()
			: Promise.resolve({
					success: true as const,
					code: 200,
					message: "",
					data: [] as PortalAppointmentRequest[],
				}),
	);
	const list = state.status === "ready" ? state.data : [];
	return {
		pending: list.filter((r) => r.status === "pending").sort(byStart),
		closed: list.filter((r) => CLOSED_REQUEST_STATUS.includes(r.status)),
		reload,
	};
}

export function AppointmentsSection({ clinic }: { clinic: Me["clinic"] }) {
	const { textGet } = useText();
	const booking = Boolean(clinic.booking?.enabled);
	const [state, reloadAppointments] = useReloadableLoad(getAppointments);
	const requests = useRequests(booking);
	const [target, setTarget] = React.useState<CancelTarget>();
	// The dialog keeps its texts while it animates closed (target is gone then).
	const [dialogKind, setDialogKind] = React.useState<CancelTarget["kind"]>();
	if (target && target.kind !== dialogKind) setDialogKind(target.kind);

	const onCancelAppointment = (appointment: PortalAppointment) =>
		setTarget({ kind: "appointment", appointment });
	const onCancelRequest = (request: PortalAppointmentRequest) =>
		setTarget({ kind: "request", request });

	const confirmCancel = async () => {
		if (!target) return false;
		const res =
			target.kind === "appointment"
				? await cancelAppointment(target.appointment.id)
				: await cancelBookingRequest(target.request.id);
		if (res.success) {
			if (target.kind === "appointment") reloadAppointments();
			else requests.reload();
		}
		return res.success;
	};

	return (
		<div className="space-y-8">
			{booking && (
				<Link
					to={`/${clinic.slug}/citas/nueva`}
					className={cn(
						buttonVariants(),
						"h-12 w-full gap-2 rounded-full text-base sm:w-auto sm:px-6 lg:hidden",
					)}
				>
					<CalendarPlus className="size-5" />
					{textGet("portal.booking.new")}
				</Link>
			)}

			{requests.pending.length > 0 && (
				<section>
					<h2 className="text-base font-semibold">
						{textGet("portal.booking.pending_title")}
					</h2>
					<Ledger>
						{requests.pending.map((r) => (
							<PendingRequestRow
								key={r.id}
								request={r}
								onCancel={onCancelRequest}
							/>
						))}
					</Ledger>
				</section>
			)}

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
								<NextAppointment
									appointment={next}
									booking={booking}
									onCancel={onCancelAppointment}
								/>
							) : (
								<div className="flex items-start gap-3 rounded-3xl border border-dashed px-5 py-5">
									<CalendarClock className="mt-0.5 size-5 shrink-0 text-primary" />
									<p className="text-sm text-muted-foreground">
										{textGet(
											booking
												? "portal.appointments.none_upcoming_bookable"
												: "portal.appointments.none_upcoming",
										)}
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
											<AppointmentRow
												key={a.id}
												appointment={a}
												past={false}
												booking={booking}
												onCancel={onCancelAppointment}
											/>
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
													<AppointmentRow
														key={a.id}
														appointment={a}
														past
														booking={booking}
														onCancel={onCancelAppointment}
													/>
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

			{requests.closed.length > 0 && (
				<section>
					<h2 className="text-base font-semibold text-muted-foreground">
						{textGet("portal.booking.closed_title")}
					</h2>
					<Ledger>
						{requests.closed.map((r) => (
							<ClosedRequestRow key={r.id} request={r} />
						))}
					</Ledger>
				</section>
			)}

			<ConfirmDialog
				open={Boolean(target)}
				onOpenChange={(open) => !open && setTarget(undefined)}
				title={textGet(
					dialogKind === "request"
						? "portal.booking.cancel_request.title"
						: "portal.booking.cancel_appointment.title",
				)}
				description={textGet(
					dialogKind === "request"
						? "portal.booking.cancel_request.body"
						: "portal.booking.cancel_appointment.body",
				)}
				confirmLabel={textGet(
					dialogKind === "request"
						? "portal.booking.cancel_request"
						: "portal.booking.cancel_appointment",
				)}
				onConfirm={confirmCancel}
			/>
		</div>
	);
}
