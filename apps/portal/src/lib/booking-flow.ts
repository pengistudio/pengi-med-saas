import {
	type ErrorResponse,
	parseDateOnly,
	toDateOnlyString,
} from "@pengi/shared";
import {
	BOOKING_ERROR,
	type BookingDoctor,
	type BookingOptions,
	type BookingSpecialty,
	type BookingTarget,
	type BookingType,
} from "@/api/portal-service";

/**
 * The booking flow's state is only what the patient picked; the step on
 * screen follows from it. Steps with a single option are picked for them
 * and never shown.
 */

export type Provider =
	| { kind: "doctor"; key: string; doctor: BookingDoctor }
	| { kind: "specialty"; key: string; specialty: BookingSpecialty };

export interface Selection {
	provider?: Provider;
	type?: BookingType;
	/** YYYY-MM-DD and "HH:MM" of the chosen slot. */
	date?: string;
	time?: string;
}

export type Step = "provider" | "type" | "when" | "review";

/**
 * What the patient chooses first: doctors, or specialties (by `choose_by`).
 * Only those that offer at least one appointment type.
 */
export function providersOf(options: BookingOptions): Provider[] {
	return allProviders(options).filter((p) => typesFor(options, p).length > 0);
}

function allProviders(options: BookingOptions): Provider[] {
	if (options.choose_by === "specialty") {
		return options.specialties.map((specialty) => ({
			kind: "specialty",
			key: `s:${specialty.specialty}|${specialty.specialty_other}`,
			specialty,
		}));
	}
	return (options.doctors ?? []).map((doctor) => ({
		kind: "doctor",
		key: `d:${doctor.id}`,
		doctor,
	}));
}

/** The appointment types the provider offers (for a specialty, any of its doctors'). */
export function typesFor(
	options: BookingOptions,
	provider: Provider,
): BookingType[] {
	const ids = new Set(
		provider.kind === "doctor"
			? provider.doctor.type_ids
			: provider.specialty.type_ids,
	);
	return options.types.filter((t) => ids.has(t.id));
}

/** The API's slot filter for a provider. */
export function targetOf(provider: Provider): BookingTarget {
	return provider.kind === "doctor"
		? { doctor_id: provider.doctor.id }
		: {
				specialty: provider.specialty.specialty,
				specialty_other: provider.specialty.specialty_other,
			};
}

/** Picks the provider, and its type when it offers only one. */
export function selectProvider(
	options: BookingOptions,
	provider: Provider,
): Selection {
	const types = typesFor(options, provider);
	return { provider, type: types.length === 1 ? types[0] : undefined };
}

/** The selection to start with: whatever has a single option is already picked. */
export function initialSelection(options: BookingOptions): Selection {
	const providers = providersOf(options);
	return providers.length === 1 ? selectProvider(options, providers[0]) : {};
}

/** Whether the patient can book at all: some provider offers some type. */
export function canBook(options: BookingOptions): boolean {
	return providersOf(options).length > 0;
}

/** The step on screen for a selection. */
export function currentStep(selection: Selection): Step {
	if (!selection.provider) return "provider";
	if (!selection.type) return "type";
	if (!selection.date || !selection.time) return "when";
	return "review";
}

/**
 * The steps the patient sees, in order. Before a provider is picked the type
 * step counts when the clinic has more than one type.
 */
export function visibleSteps(
	options: BookingOptions,
	selection: Selection,
): Step[] {
	const steps: Step[] = [];
	if (providersOf(options).length > 1) steps.push("provider");
	const types = selection.provider
		? typesFor(options, selection.provider)
		: options.types;
	if (types.length > 1) steps.push("type");
	steps.push("when", "review");
	return steps;
}

/**
 * The selection one step back, or undefined when the first visible step is
 * on screen (going back leaves the flow).
 */
export function stepBack(
	options: BookingOptions,
	selection: Selection,
): Selection | undefined {
	const steps = visibleSteps(options, selection);
	const index = steps.indexOf(currentStep(selection));
	if (index <= 0) return undefined;
	switch (steps[index - 1]) {
		case "provider":
			return {};
		case "type":
			return { provider: selection.provider };
		case "when":
			return { ...selection, time: undefined };
		default:
			return selection;
	}
}

// ─── Slot pages ──────────────────────────────────────────────────────────────

/** Days fetched per request (the API's limit). */
export const SLOT_PAGE_DAYS = 14;

export function addDays(date: string, days: number): string {
	const parsed = parseDateOnly(date) ?? new Date();
	parsed.setDate(parsed.getDate() + days);
	return toDateOnlyString(parsed);
}

/** The next page of days from `from`, clamped to the window; undefined past it. */
export function slotRange(
	from: string,
	maxDate: string,
): { from: string; to: string } | undefined {
	if (from > maxDate) return undefined;
	const to = addDays(from, SLOT_PAGE_DAYS - 1);
	return { from, to: to > maxDate ? maxDate : to };
}

// ─── Errors ──────────────────────────────────────────────────────────────────

export type BookingErrorKind = "slot_taken" | "too_many_pending" | "other";

/** Why a booking request failed, from its error code or HTTP status. */
export function bookingErrorKind(res: ErrorResponse): BookingErrorKind {
	const code = res.data?.error_code;
	if (code === BOOKING_ERROR.slotTaken) return "slot_taken";
	if (code === BOOKING_ERROR.tooManyPending) return "too_many_pending";
	if (res.code === 429) return "too_many_pending";
	if (res.code === 409) return "slot_taken";
	return "other";
}
