import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { Pencil } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	type BookingOptions,
	type Clinic,
	createNewPatientRequest,
	getBookingOptions,
	guestSlotsLoader,
	type NewPatientRequestResult,
} from "@/api/portal-service";
import {
	BookingOptionsState,
	BookingPicker,
	BookingShell,
	pickStepTitleKey,
	ReasonField,
	ResultPanel,
	SelectionCard,
	StepAlert,
	StepFrame,
	SubmitButton,
	useBookingSelection,
	useSelectionSummary,
	useStepFocus,
} from "@/components/booking/booking-flow";
import { CodeForm, ContactForm } from "@/components/new-patient/contact-step";
import { DetailsForm } from "@/components/new-patient/details-form";
import { canBook, targetOf } from "@/lib/booking-flow";
import {
	contactChannels,
	initialPatientValues,
	type NewPatientErrorKind,
	type NewPatientValues,
	newPatientErrorKind,
	newPatientSteps,
	toPatientData,
	type VerifiedContact,
} from "@/lib/new-patient-flow";
import { useLoad } from "@/lib/use-load";

type Phase = "contact" | "code" | "booking" | "not_accepted";

/**
 * `/:slug/primera-cita`: a person without a record verifies a contact, picks
 * a slot, gives their data and sends the request. The guest token lives only
 * here (memory): a reload starts over.
 */
export function NewPatientFlow({ clinic }: { clinic: Clinic }) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const signIn = `/${clinic.slug}/entrar`;
	const [phase, setPhase] = React.useState<Phase>("contact");
	const [verified, setVerified] = React.useState<VerifiedContact>();
	const [guestToken, setGuestToken] = React.useState<string>();
	const [expired, setExpired] = React.useState(false);
	const headingRef = useStepFocus(phase);

	const restart = (tokenExpired = false) => {
		setGuestToken(undefined);
		setExpired(tokenExpired);
		setPhase("contact");
	};

	if (phase === "booking" && guestToken && verified) {
		return (
			<GuestOptions
				key={guestToken}
				guestToken={guestToken}
				verified={verified}
				signIn={signIn}
				onLeave={() => restart()}
				onExpired={() => restart(true)}
			/>
		);
	}

	if (phase === "not_accepted") {
		return (
			<BookingShell onBack={signIn}>
				<div className="space-y-2 pt-6">
					<h1
						ref={headingRef}
						tabIndex={-1}
						className="text-2xl font-semibold tracking-tight outline-none"
					>
						{textGet("portal.new_patient.not_accepted.title")}
					</h1>
					<p className="text-muted-foreground">
						{textGet("portal.new_patient.not_accepted.body")}
					</p>
				</div>
			</BookingShell>
		);
	}

	const onCode = phase === "code" && verified;

	return (
		<BookingShell
			wide
			onBack={onCode ? () => setPhase("contact") : () => navigate(signIn)}
		>
			<StepFrame
				title={textGet(
					onCode
						? "portal.new_patient.step.code"
						: "portal.new_patient.step.contact",
				)}
				summary={textGet("portal.new_patient.intro", { clinic: clinic.name })}
				headingRef={headingRef}
			>
				{expired && !onCode && (
					<StepAlert>{textGet("portal.new_patient.error.expired")}</StepAlert>
				)}
				{onCode ? (
					<CodeForm
						slug={clinic.slug}
						contact={verified}
						onVerified={(token) => {
							setGuestToken(token);
							setExpired(false);
							setPhase("booking");
						}}
						onChangeContact={() => setPhase("contact")}
					/>
				) : (
					<ContactForm
						slug={clinic.slug}
						channels={contactChannels(clinic.channels)}
						initial={verified}
						onCodeSent={(contact) => {
							setVerified(contact);
							setPhase("code");
						}}
						onNotAccepted={() => setPhase("not_accepted")}
					/>
				)}
			</StepFrame>
		</BookingShell>
	);
}

/** Loads what the guest can book, then the booking steps. */
function GuestOptions({
	guestToken,
	verified,
	signIn,
	onLeave,
	onExpired,
}: {
	guestToken: string;
	verified: VerifiedContact;
	signIn: string;
	onLeave: () => void;
	onExpired: () => void;
}) {
	const state = useLoad(() => getBookingOptions(guestToken));
	if (state.status !== "ready" || !canBook(state.data)) {
		return (
			<BookingShell onBack={onLeave}>
				<BookingOptionsState
					status={state.status === "ready" ? "unavailable" : state.status}
				/>
			</BookingShell>
		);
	}
	return (
		<GuestBooking
			options={state.data}
			guestToken={guestToken}
			verified={verified}
			signIn={signIn}
			onLeave={onLeave}
			onExpired={onExpired}
		/>
	);
}

/** Provider → type → when (shared with patients), then data, review and result. */
export function GuestBooking({
	options,
	guestToken,
	verified,
	signIn,
	onLeave,
	onExpired,
}: {
	options: BookingOptions;
	guestToken: string;
	verified: VerifiedContact;
	signIn: string;
	/** Back from the first step: verify again. */
	onLeave: () => void;
	/** The guest token lapsed: verify again. */
	onExpired: () => void;
}) {
	const { textGet } = useText();
	const summarize = useSelectionSummary();
	const booking = useBookingSelection(options);
	const loadSlots = React.useMemo(
		() => guestSlotsLoader(guestToken),
		[guestToken],
	);
	// Once the time is picked: the data, then the review.
	const [finish, setFinish] = React.useState<"details" | "review">("details");
	const [values, setValues] = React.useState<NewPatientValues>(() =>
		initialPatientValues(verified),
	);
	const [reason, setReason] = React.useState("");
	const [submitting, setSubmitting] = React.useState(false);
	const [error, setError] = React.useState<{
		kind: NewPatientErrorKind;
		message: string;
	}>();
	const [result, setResult] = React.useState<NewPatientRequestResult>();

	const { selection } = booking;
	const picking = booking.step !== "review";
	const step = picking ? booking.step : finish;
	const steps = newPatientSteps(booking.steps);
	const headingRef = useStepFocus(result ? "result" : step);

	const goBack = () => {
		setError(undefined);
		if (step === "review") setFinish("details");
		else if (!booking.back()) onLeave();
	};

	const submit = async () => {
		const { provider, type, date, time } = selection;
		if (!provider || !type || !date || !time) return;
		setSubmitting(true);
		setError(undefined);
		const trimmed = reason.trim();
		const res = await createNewPatientRequest(guestToken, {
			type_id: type.id,
			date,
			start_time: time,
			...targetOf(provider),
			...(trimmed ? { reason: trimmed } : {}),
			patient: toPatientData(values, verified),
		});
		setSubmitting(false);
		if (res.success) {
			setResult(res.data);
			return;
		}
		const kind = newPatientErrorKind(res);
		if (kind === "expired") {
			onExpired();
			return;
		}
		setError({ kind, message: res.message });
		if (kind === "slot_taken") booking.slotTaken();
		if (kind === "invalid_data") setFinish("details");
	};

	if (result) {
		return (
			<BookingShell onBack={signIn}>
				<ResultPanel
					confirmed={false}
					title={textGet("portal.new_patient.done.title")}
					body={textGet(
						result.contact_channel === "whatsapp"
							? "portal.new_patient.done.body.whatsapp"
							: "portal.new_patient.done.body.email",
					)}
					request={result}
					home={signIn}
					backLabel={textGet("portal.new_patient.done.back")}
					headingRef={headingRef}
				/>
			</BookingShell>
		);
	}

	const { provider, type, date } = selection;
	const title =
		step === "details" || step === "review"
			? textGet(
					step === "details"
						? "portal.new_patient.step.details"
						: "portal.booking.step.review",
				)
			: textGet(pickStepTitleKey(options, step));

	return (
		<BookingShell onBack={goBack} wide>
			<StepFrame
				position={steps.indexOf(step) + 1}
				total={steps.length}
				title={title}
				summary={
					provider &&
					step !== "provider" &&
					step !== "review" &&
					summarize(provider, type)
				}
				headingRef={headingRef}
			>
				{error && (
					<StepAlert>
						{error.kind === "other"
							? error.message
							: textGet(`portal.new_patient.error.${error.kind}`)}
					</StepAlert>
				)}

				{picking && (
					<BookingPicker
						options={options}
						booking={booking}
						loadSlots={loadSlots}
						// After "slot taken", data already given: straight back to the review.
						onPicked={() => setError(undefined)}
					/>
				)}

				{step === "details" && (
					<DetailsForm
						verified={verified}
						initialValues={values}
						onSubmit={(next) => {
							setValues(next);
							setError(undefined);
							setFinish("review");
						}}
					/>
				)}

				{step === "review" && provider && type && date && (
					<form
						className="space-y-6"
						onSubmit={(e) => {
							e.preventDefault();
							submit();
						}}
					>
						<SelectionCard selection={{ ...selection, provider, type, date }} />
						<PatientSummary
							values={values}
							verified={verified}
							onEdit={() => {
								setError(undefined);
								setFinish("details");
							}}
						/>
						<ReasonField reason={reason} onReason={setReason} />
						<SubmitButton submitting={submitting} />
					</form>
				)}
			</StepFrame>
		</BookingShell>
	);
}

/** The new patient's data as it will reach the clinic, with a way back to edit it. */
function PatientSummary({
	values,
	verified,
	onEdit,
}: {
	values: NewPatientValues;
	verified: VerifiedContact;
	onEdit: () => void;
}) {
	const { textGet, formatDate } = useText();
	const data = toPatientData(values, verified);
	const rows: [string, string][] = [
		[
			textGet(
				data.document_type === "passport"
					? "portal.new_patient.document.passport"
					: "portal.new_patient.document.cedula",
			),
			data.document,
		],
		[
			textGet("portal.new_patient.birth_date"),
			formatDate(data.birth_date, "long"),
		],
	];
	if (data.phone) rows.push([textGet("portal.new_patient.phone"), data.phone]);
	if (data.email) rows.push([textGet("portal.new_patient.email"), data.email]);

	return (
		<section
			aria-labelledby="new-patient-summary"
			className="rounded-3xl border bg-card px-5 py-5"
		>
			<div className="flex items-start justify-between gap-3">
				<div className="min-w-0">
					<h2
						id="new-patient-summary"
						className="text-sm text-muted-foreground"
					>
						{textGet("portal.new_patient.summary")}
					</h2>
					<p className="mt-0.5 text-lg font-semibold break-words">
						{data.first_name} {data.last_name}
					</p>
				</div>
				<Button
					type="button"
					variant="ghost"
					className="-mr-2 h-10 shrink-0 gap-1.5 rounded-full px-3"
					onClick={onEdit}
				>
					<Pencil className="size-4" />
					{textGet("portal.new_patient.edit")}
				</Button>
			</div>
			<dl className="mt-3 grid gap-2 text-sm sm:grid-cols-2">
				{rows.map(([label, value]) => (
					<div key={label} className="min-w-0">
						<dt className="text-muted-foreground">{label}</dt>
						<dd className="break-words">{value}</dd>
					</div>
				))}
			</dl>
		</section>
	);
}
