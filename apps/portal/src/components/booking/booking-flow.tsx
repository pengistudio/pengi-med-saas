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
	createBookingRequest,
	getBookingOptions,
	type Me,
	type PortalAppointmentRequest,
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

/** `/:slug/citas/nueva`: who, what, when, review, done. */
export function BookingFlow({ clinic }: { clinic: Me["clinic"] }) {
	const { textGet } = useText();
	const state = useLoad(getBookingOptions);
	const home = `/${clinic.slug}`;

	if (state.status === "loading") {
		return (
			<BookingShell onBack={home}>
				<div className="space-y-3 pt-6">
					<Skeleton className="h-8 w-2/3" />
					<Skeleton className="h-18 w-full rounded-2xl" />
					<Skeleton className="h-18 w-full rounded-2xl" />
				</div>
			</BookingShell>
		);
	}
	if (state.status === "error" || !canBook(state.data)) {
		return (
			<BookingShell onBack={home}>
				<p className="pt-6 text-sm text-muted-foreground">
					{textGet(
						state.status === "error"
							? "portal.section.error"
							: "portal.booking.unavailable",
					)}
				</p>
			</BookingShell>
		);
	}
	return <BookingSteps options={state.data} home={home} />;
}

/** Page frame: a back button and a centered column (wider for the steps). */
function BookingShell({
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

export function BookingSteps({
	options,
	home,
}: {
	options: BookingOptions;
	home: string;
}) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const providerText = useProviderText();
	const [selection, setSelection] = React.useState<Selection>(() =>
		initialSelection(options),
	);
	const [reason, setReason] = React.useState("");
	const [submitting, setSubmitting] = React.useState(false);
	const [error, setError] = React.useState<{
		kind: BookingErrorKind;
		message: string;
	}>();
	// Bumped to fetch the slots again (after "slot taken").
	const [slotsVersion, setSlotsVersion] = React.useState(0);
	const [result, setResult] = React.useState<PortalAppointmentRequest>();

	const step = currentStep(selection);
	const steps = visibleSteps(options, selection);
	const headingRef = React.useRef<HTMLHeadingElement>(null);

	// A new step is a new screen: start at its top, focus its title.
	React.useEffect(() => {
		window.scrollTo?.({ top: 0 });
		headingRef.current?.focus();
	}, [step, result]);

	const goBack = () => {
		const previous = stepBack(options, selection);
		if (!previous) navigate(home);
		else {
			setSelection(previous);
			setError(undefined);
		}
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
		if (kind === "slot_taken") {
			// Back to the times, fetched again without the one just taken.
			setSelection((s) => ({ ...s, time: undefined }));
			setSlotsVersion((v) => v + 1);
		}
	};

	if (result) {
		return (
			<BookingShell onBack={home}>
				<BookingResult request={result} home={home} headingRef={headingRef} />
			</BookingShell>
		);
	}

	const { provider, type } = selection;
	const position = steps.indexOf(step) + 1;
	const titleKey =
		step === "provider"
			? `portal.booking.step.provider.${options.choose_by}`
			: `portal.booking.step.${step}`;

	return (
		<BookingShell onBack={goBack} wide>
			<div className="lg:grid lg:grid-cols-[17rem_minmax(0,1fr)] lg:gap-16 lg:pt-6">
				<div className="space-y-1 pb-6 lg:sticky lg:top-8 lg:self-start">
					<p className="text-sm text-muted-foreground">
						{textGet("portal.booking.step_of", {
							current: position,
							total: steps.length,
						})}
					</p>
					<div aria-hidden="true" className="flex gap-1 pb-2">
						{steps.map((s, i) => (
							<span
								key={s}
								className={cn(
									"h-1 flex-1 rounded-full",
									i < position ? "bg-primary" : "bg-muted",
								)}
							/>
						))}
					</div>
					<h1
						ref={headingRef}
						tabIndex={-1}
						className="text-2xl font-semibold tracking-tight outline-none"
					>
						{textGet(titleKey)}
					</h1>
					{provider && step !== "provider" && step !== "review" && (
						<p className="text-sm text-muted-foreground lg:pt-2 lg:text-[15px]">
							{[
								providerText(provider).title,
								type &&
									`${type.name} · ${textGet("portal.booking.minutes", { count: type.duration_minutes })}`,
							]
								.filter(Boolean)
								.join(" · ")}
						</p>
					)}
				</div>

				<div className="min-w-0">
					{error && (
						<p
							role="alert"
							className="mb-5 rounded-2xl bg-destructive/10 px-4 py-3 text-sm text-destructive"
						>
							{error.kind === "other"
								? error.message
								: textGet(`portal.booking.error.${error.kind}`)}
						</p>
					)}

					{step === "provider" && (
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
					)}

					{step === "type" && provider && (
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
					)}

					{step === "when" && provider && type && (
						<SlotPicker
							key={`${provider.key}-${type.id}-${slotsVersion}`}
							typeId={type.id}
							target={targetOf(provider)}
							minDate={options.min_date}
							maxDate={options.max_date}
							initialDate={selection.date}
							onPick={(date, time) => {
								setError(undefined);
								setSelection({ provider, type, date, time });
							}}
						/>
					)}

					{step === "review" && provider && type && selection.date && (
						<ReviewStep
							selection={{ ...selection, provider, type, date: selection.date }}
							reason={reason}
							onReason={setReason}
							submitting={submitting}
							onSubmit={submit}
						/>
					)}
				</div>
			</div>
		</BookingShell>
	);
}

function ReviewStep({
	selection,
	reason,
	onReason,
	submitting,
	onSubmit,
}: {
	selection: Required<Pick<Selection, "provider" | "type" | "date">> &
		Selection;
	reason: string;
	onReason: (reason: string) => void;
	submitting: boolean;
	onSubmit: () => void;
}) {
	const { textGet, formatDate } = useText();
	const providerText = useProviderText();
	const { provider, type, date, time } = selection;
	const who = providerText(provider);

	return (
		<form
			className="space-y-6"
			onSubmit={(e) => {
				e.preventDefault();
				onSubmit();
			}}
		>
			<section className="rounded-3xl bg-secondary px-5 py-6 text-secondary-foreground">
				<p className="text-3xl font-semibold tracking-tight text-foreground first-letter:uppercase">
					{formatDate(date, "weekday-day-month")}
				</p>
				<p className="text-3xl font-semibold tracking-tight text-primary tabular-nums">
					{time}
				</p>
				<div className="mt-5 space-y-0.5 text-foreground">
					{provider.kind === "doctor" ? (
						<>
							<p className="font-medium">{who.title}</p>
							{who.subtitle && (
								<p className="text-muted-foreground">{who.subtitle}</p>
							)}
						</>
					) : (
						<>
							<p className="font-medium">{who.title}</p>
							<p className="text-muted-foreground">
								{textGet("portal.booking.any_doctor")}
							</p>
						</>
					)}
					<p className="text-muted-foreground">
						{type.name} ·{" "}
						{textGet("portal.booking.minutes", {
							count: type.duration_minutes,
						})}
					</p>
				</div>
			</section>

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

			<Button
				type="submit"
				disabled={submitting}
				className="h-12 w-full gap-2 rounded-full text-base"
			>
				{submitting ? <Spinner /> : <CalendarCheck className="size-5" />}
				{textGet("portal.booking.submit")}
			</Button>
		</form>
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
	const { textGet, formatDate } = useText();
	const confirmed = request.status === "approved";
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
					{textGet(
						confirmed
							? "portal.booking.done.confirmed.title"
							: "portal.booking.done.pending.title",
					)}
				</h1>
				<p className="text-muted-foreground">
					{textGet(
						confirmed
							? "portal.booking.done.confirmed.body"
							: "portal.booking.done.pending.body",
					)}
				</p>
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
				{textGet("portal.booking.done.back")}
			</Link>
		</div>
	);
}
