import { useText } from "@pengi/shared";
import {
	Button,
	buttonVariants,
	cn,
	Skeleton,
	Spinner,
	Textarea,
} from "@pengi/ui";
import {
	ArrowLeft,
	CalendarCheck,
	Clock,
	Hourglass,
	Stethoscope,
} from "lucide-react";
import React from "react";
import { Link, useNavigate } from "react-router";
import {
	type BookingOptions,
	type BookingType,
	createBookingRequest,
	getBookingOptions,
	type Me,
	type PortalAppointmentRequest,
	type SlotsLoader,
} from "@/api/portal-service";
import { OptionButton, OptionList } from "@/components/booking/option-list";
import { SlotPicker } from "@/components/booking/slot-picker";
import {
	type BookingErrorKind,
	bookingErrorKind,
	canBook,
	currentStep,
	initialSelection,
	type Provider,
	providersOf,
	type Selection,
	type Step,
	selectProvider,
	stepBack,
	targetOf,
	typesFor,
	visibleSteps,
} from "@/lib/booking-flow";
import { nameCase } from "@/lib/person-name";
import { useSpecialtyLabel } from "@/lib/specialty";
import { useLoad } from "@/lib/use-load";

export const REASON_MAX = 500;

const initials = (name: string) =>
	name
		.replace(/^(dra?|dr)\.?\s+/i, "")
		.split(/\s+/)
		.filter(Boolean)
		.slice(0, 2)
		.map((word) => word[0]?.toUpperCase())
		.join("");

function useProviderText() {
	const specialtyLabel = useSpecialtyLabel();
	return (provider: Provider) =>
		provider.kind === "doctor"
			? {
					title: nameCase(provider.doctor.full_name),
					subtitle: specialtyLabel(provider.doctor) || undefined,
				}
			: { title: specialtyLabel(provider.specialty), subtitle: undefined };
}

/** "Dra. Ana Paz · Consulta · 30 min": what is picked so far, for the step aside. */
export function useSelectionSummary() {
	const { textGet } = useText();
	const providerText = useProviderText();
	return (provider: Provider, type?: BookingType) =>
		[
			providerText(provider).title,
			type &&
				`${type.name} · ${textGet("portal.booking.minutes", { count: type.duration_minutes })}`,
		]
			.filter(Boolean)
			.join(" · ");
}

/** `/:slug/citas/nueva`: who, what, when, review, done. */
export function BookingFlow({ clinic }: { clinic: Me["clinic"] }) {
	const state = useLoad(getBookingOptions);
	const home = `/${clinic.slug}`;

	if (state.status !== "ready" || !canBook(state.data)) {
		return (
			<BookingShell onBack={home}>
				<BookingOptionsState
					status={state.status === "ready" ? "unavailable" : state.status}
				/>
			</BookingShell>
		);
	}
	return <BookingSteps options={state.data} home={home} />;
}

/** While the booking options load, or why there is nothing to book. */
export function BookingOptionsState({
	status,
}: {
	status: "loading" | "error" | "unavailable";
}) {
	const { textGet } = useText();
	if (status === "loading") {
		return (
			<div className="space-y-3 pt-6">
				<Skeleton className="h-8 w-2/3" />
				<Skeleton className="h-18 w-full rounded-2xl" />
				<Skeleton className="h-18 w-full rounded-2xl" />
			</div>
		);
	}
	return (
		<p className="pt-6 text-sm text-muted-foreground">
			{textGet(
				status === "error"
					? "portal.section.error"
					: "portal.booking.unavailable",
			)}
		</p>
	);
}

/** Page frame: a back button and a centered column (wider for the steps). */
export function BookingShell({
	onBack,
	wide = false,
	children,
}: {
	/** A path, or a handler for going one step back. */
	onBack: string | (() => void);
	/** From `lg`, room for the step aside next to its content. */
	wide?: boolean;
	children: React.ReactNode;
}) {
	const { textGet } = useText();
	const navigate = useNavigate();
	return (
		<div
			className={cn(
				"mx-auto min-h-screen w-full max-w-xl px-4 pb-16",
				wide && "lg:max-w-5xl lg:px-10",
			)}
		>
			<header className="py-4">
				<Button
					type="button"
					variant="ghost"
					className="-ml-2 h-10 gap-1.5 rounded-full px-3 text-muted-foreground"
					onClick={() =>
						typeof onBack === "string" ? navigate(onBack) : onBack()
					}
				>
					<ArrowLeft className="size-5" />
					{textGet("portal.booking.back")}
				</Button>
			</header>
			{children}
		</div>
	);
}

/**
 * One step of a flow: progress, title and what is picked so far, beside the
 * step's content from `lg` (stacked on a phone).
 */
export function StepFrame({
	position,
	total,
	title,
	summary,
	headingRef,
	children,
}: {
	/** 1-based; without it (and `total`) no progress is shown. */
	position?: number;
	total?: number;
	title: string;
	summary?: React.ReactNode;
	headingRef: React.RefObject<HTMLHeadingElement | null>;
	children: React.ReactNode;
}) {
	const { textGet } = useText();
	return (
		<div className="lg:grid lg:grid-cols-[17rem_minmax(0,1fr)] lg:gap-16 lg:pt-6">
			<div className="space-y-1 pb-6 lg:sticky lg:top-8 lg:self-start">
				{position !== undefined && total !== undefined && (
					<>
						<p className="text-sm text-muted-foreground">
							{textGet("portal.booking.step_of", { current: position, total })}
						</p>
						<div aria-hidden="true" className="flex gap-1 pb-2">
							{Array.from({ length: total }, (_, i) => (
								<span
									key={i}
									className={cn(
										"h-1 flex-1 rounded-full",
										i < position ? "bg-primary" : "bg-muted",
									)}
								/>
							))}
						</div>
					</>
				)}
				<h1
					ref={headingRef}
					tabIndex={-1}
					className="text-2xl font-semibold tracking-tight outline-none"
				>
					{title}
				</h1>
				{summary && (
					<p className="text-sm text-muted-foreground lg:pt-2 lg:text-[15px]">
						{summary}
					</p>
				)}
			</div>
			<div className="min-w-0">{children}</div>
		</div>
	);
}

/** A new step is a new screen: start at its top, focus its title. */
export function useStepFocus(key: unknown) {
	const headingRef = React.useRef<HTMLHeadingElement>(null);
	React.useEffect(() => {
		window.scrollTo?.({ top: 0 });
		headingRef.current?.focus();
	}, [key]);
	return headingRef;
}

/** An error above a step's content. */
export function StepAlert({ children }: { children: React.ReactNode }) {
	return (
		<p
			role="alert"
			className="mb-5 rounded-2xl bg-destructive/10 px-4 py-3 text-sm text-destructive"
		>
			{children}
		</p>
	);
}

/**
 * The picking part of booking (provider → type → when), shared by patients
 * and new patients: what is picked, the step it leads to, going back, and
 * fetching the times again after a slot was taken.
 */
export function useBookingSelection(options: BookingOptions) {
	const [selection, setSelection] = React.useState<Selection>(() =>
		initialSelection(options),
	);
	// Bumped to fetch the slots again (after "slot taken").
	const [slotsVersion, setSlotsVersion] = React.useState(0);

	return {
		selection,
		setSelection,
		slotsVersion,
		step: currentStep(selection),
		steps: visibleSteps(options, selection),
		/** One step back; false when on the first booking step. */
		back: () => {
			const previous = stepBack(options, selection);
			if (!previous) return false;
			setSelection(previous);
			return true;
		},
		/** Back to the times, fetched again without the one just taken. */
		slotTaken: () => {
			setSelection((s) => ({ ...s, time: undefined }));
			setSlotsVersion((v) => v + 1);
		},
	};
}

export type BookingSelection = ReturnType<typeof useBookingSelection>;

/** The title key of a picking step. */
export const pickStepTitleKey = (options: BookingOptions, step: Step) =>
	step === "provider"
		? `portal.booking.step.provider.${options.choose_by}`
		: `portal.booking.step.${step}`;

/** The content of the provider, type and when steps. */
export function BookingPicker({
	options,
	booking,
	loadSlots,
	onPicked,
}: {
	options: BookingOptions;
	booking: BookingSelection;
	/** Where the times come from (default: the patient's session). */
	loadSlots?: SlotsLoader;
	/** A time was picked. */
	onPicked?: () => void;
}) {
	const { textGet } = useText();
	const providerText = useProviderText();
	const { selection, setSelection, step, slotsVersion } = booking;
	const { provider, type } = selection;

	if (step === "provider") {
		return (
			<OptionList>
				{providersOf(options).map((p) => {
					const text = providerText(p);
					return (
						<OptionButton
							key={p.key}
							leading={
								p.kind === "doctor" ? (
									initials(p.doctor.full_name)
								) : (
									<Stethoscope className="size-5" />
								)
							}
							title={text.title}
							subtitle={text.subtitle}
							onClick={() => setSelection(selectProvider(options, p))}
						/>
					);
				})}
			</OptionList>
		);
	}
	if (step === "type" && provider) {
		return (
			<OptionList>
				{typesFor(options, provider).map((t) => (
					<OptionButton
						key={t.id}
						leading={<Clock className="size-5" />}
						title={t.name}
						subtitle={textGet("portal.booking.minutes", {
							count: t.duration_minutes,
						})}
						onClick={() => setSelection({ provider, type: t })}
					/>
				))}
			</OptionList>
		);
	}
	if (step === "when" && provider && type) {
		return (
			<SlotPicker
				key={`${provider.key}-${type.id}-${slotsVersion}`}
				typeId={type.id}
				target={targetOf(provider)}
				minDate={options.min_date}
				maxDate={options.max_date}
				initialDate={selection.date}
				loadSlots={loadSlots}
				onPick={(date, time) => {
					setSelection({ provider, type, date, time });
					onPicked?.();
				}}
			/>
		);
	}
	return null;
}

export function BookingSteps({
	options,
	home,
}: {
	options: BookingOptions;
	home: string;
}) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const summarize = useSelectionSummary();
	const booking = useBookingSelection(options);
	const [reason, setReason] = React.useState("");
	const [submitting, setSubmitting] = React.useState(false);
	const [error, setError] = React.useState<{
		kind: BookingErrorKind;
		message: string;
	}>();
	const [result, setResult] = React.useState<PortalAppointmentRequest>();

	const { selection, step, steps } = booking;
	const headingRef = useStepFocus(result ? "result" : step);

	const goBack = () => {
		if (!booking.back()) navigate(home);
		else setError(undefined);
	};

	const submit = async () => {
		const { provider, type, date, time } = selection;
		if (!provider || !type || !date || !time) return;
		setSubmitting(true);
		setError(undefined);
		const trimmed = reason.trim();
		const res = await createBookingRequest({
			type_id: type.id,
			date,
			start_time: time,
			...targetOf(provider),
			...(trimmed ? { reason: trimmed } : {}),
		});
		setSubmitting(false);
		if (res.success) {
			setResult(res.data);
			return;
		}
		const kind = bookingErrorKind(res);
		setError({ kind, message: res.message });
		if (kind === "slot_taken") booking.slotTaken();
	};

	if (result) {
		return (
			<BookingShell onBack={home}>
				<BookingResult request={result} home={home} headingRef={headingRef} />
			</BookingShell>
		);
	}

	const { provider, type, date } = selection;

	return (
		<BookingShell onBack={goBack} wide>
			<StepFrame
				position={steps.indexOf(step) + 1}
				total={steps.length}
				title={textGet(pickStepTitleKey(options, step))}
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
							: textGet(`portal.booking.error.${error.kind}`)}
					</StepAlert>
				)}

				{step === "review" && provider && type && date ? (
					<form
						className="space-y-6"
						onSubmit={(e) => {
							e.preventDefault();
							submit();
						}}
					>
						<SelectionCard selection={{ ...selection, provider, type, date }} />
						<ReasonField reason={reason} onReason={setReason} />
						<SubmitButton submitting={submitting} />
					</form>
				) : (
					<BookingPicker
						options={options}
						booking={booking}
						onPicked={() => setError(undefined)}
					/>
				)}
			</StepFrame>
		</BookingShell>
	);
}

/** The chosen day, time, provider and type, large. */
export function SelectionCard({
	selection,
	children,
}: {
	selection: Required<Pick<Selection, "provider" | "type" | "date">> &
		Selection;
	/** More lines under the appointment (e.g. the new patient's data). */
	children?: React.ReactNode;
}) {
	const { textGet, formatDate } = useText();
	const providerText = useProviderText();
	const { provider, type, date, time } = selection;
	const who = providerText(provider);

	return (
		<section className="rounded-3xl bg-secondary px-5 py-6 text-secondary-foreground">
			<p className="text-3xl font-semibold tracking-tight text-foreground first-letter:uppercase">
				{formatDate(date, "weekday-day-month")}
			</p>
			<p className="text-3xl font-semibold tracking-tight text-primary tabular-nums">
				{time}
			</p>
			<div className="mt-5 space-y-0.5 text-foreground">
				<p className="font-medium">{who.title}</p>
				{provider.kind === "doctor" ? (
					who.subtitle && (
						<p className="text-muted-foreground">{who.subtitle}</p>
					)
				) : (
					<p className="text-muted-foreground">
						{textGet("portal.booking.any_doctor")}
					</p>
				)}
				<p className="text-muted-foreground">
					{type.name} ·{" "}
					{textGet("portal.booking.minutes", {
						count: type.duration_minutes,
					})}
				</p>
			</div>
			{children}
		</section>
	);
}

/** The optional reason for the visit. */
export function ReasonField({
	reason,
	onReason,
}: {
	reason: string;
	onReason: (reason: string) => void;
}) {
	const { textGet } = useText();
	return (
		<div className="space-y-2">
			<label htmlFor="booking-reason" className="text-sm font-medium">
				{textGet("portal.booking.reason")}{" "}
				<span className="font-normal text-muted-foreground">
					{textGet("portal.booking.optional")}
				</span>
			</label>
			<Textarea
				id="booking-reason"
				value={reason}
				maxLength={REASON_MAX}
				rows={3}
				placeholder={textGet("portal.booking.reason.placeholder")}
				onChange={(e) => onReason(e.target.value)}
				aria-describedby="booking-reason-count"
				className="rounded-2xl text-base"
			/>
			<p
				id="booking-reason-count"
				className="text-right text-xs text-muted-foreground tabular-nums"
			>
				{reason.length}/{REASON_MAX}
			</p>
		</div>
	);
}

export function SubmitButton({ submitting }: { submitting: boolean }) {
	const { textGet } = useText();
	return (
		<Button
			type="submit"
			disabled={submitting}
			className="h-12 w-full gap-2 rounded-full text-base"
		>
			{submitting ? <Spinner /> : <CalendarCheck className="size-5" />}
			{textGet("portal.booking.submit")}
		</Button>
	);
}

/** The outcome screen: an icon, a title and body, the slot, and a way out. */
export function ResultPanel({
	confirmed,
	title,
	body,
	request,
	home,
	backLabel,
	headingRef,
}: {
	confirmed: boolean;
	title: string;
	body: string;
	request: Pick<
		PortalAppointmentRequest,
		"date" | "start_time" | "doctor" | "type_name"
	>;
	home: string;
	backLabel: string;
	headingRef: React.RefObject<HTMLHeadingElement | null>;
}) {
	const { formatDate } = useText();
	const Icon = confirmed ? CalendarCheck : Hourglass;

	return (
		<div className="space-y-6 pt-4">
			<span
				aria-hidden="true"
				className="flex size-14 items-center justify-center rounded-full bg-secondary text-primary"
			>
				<Icon className="size-7" />
			</span>
			<div className="space-y-2">
				<h1
					ref={headingRef}
					tabIndex={-1}
					className="text-2xl font-semibold tracking-tight outline-none"
				>
					{title}
				</h1>
				<p className="text-muted-foreground">{body}</p>
			</div>
			<p className="rounded-3xl bg-secondary px-5 py-5 text-foreground">
				<span className="block text-xl font-semibold tracking-tight first-letter:uppercase">
					{formatDate(request.date, "weekday-day-month")} ·{" "}
					<span className="text-primary tabular-nums">
						{request.start_time.slice(0, 5)}
					</span>
				</span>
				{request.doctor?.full_name && (
					<span className="mt-1 block">
						{nameCase(request.doctor.full_name)}
					</span>
				)}
				{request.type_name && (
					<span className="block text-muted-foreground">
						{request.type_name}
					</span>
				)}
			</p>
			<Link
				to={home}
				className={cn(buttonVariants(), "h-12 w-full rounded-full text-base")}
			>
				{backLabel}
			</Link>
		</div>
	);
}

function BookingResult({
	request,
	home,
	headingRef,
}: {
	request: PortalAppointmentRequest;
	home: string;
	headingRef: React.RefObject<HTMLHeadingElement | null>;
}) {
	const { textGet } = useText();
	const confirmed = request.status === "approved";
	return (
		<ResultPanel
			confirmed={confirmed}
			title={textGet(
				confirmed
					? "portal.booking.done.confirmed.title"
					: "portal.booking.done.pending.title",
			)}
			body={textGet(
				confirmed
					? "portal.booking.done.confirmed.body"
					: "portal.booking.done.pending.body",
			)}
			request={request}
			home={home}
			backLabel={textGet("portal.booking.done.back")}
			headingRef={headingRef}
		/>
	);
}
