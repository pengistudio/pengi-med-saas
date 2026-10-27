import {
	type ErrorResponse,
	parseDateOnly,
	toDateOnlyString,
} from "@pengi/shared";
import { z } from "zod";
import {
	BOOKING_ERROR,
	type ContactChannel,
	type DocumentType,
	NEW_PATIENT_ERROR,
	type NewPatientData,
} from "@/api/portal-service";
import type { Step } from "@/lib/booking-flow";

/**
 * A new patient (no record at the clinic) verifies a contact, picks a slot
 * like a patient, gives their data and sends the request; the clinic creates
 * the record when it approves. Same rules as the API, so most mistakes are
 * caught before sending.
 */

// ─── Documents ───────────────────────────────────────────────────────────────

/**
 * An Ecuadorian cédula: 10 digits, province 01–24 or 30, third digit below 6,
 * and the modulo-10 check digit. Mirrors the API's validator.
 */
export function isValidCedula(id: string): boolean {
	if (!/^\d{10}$/.test(id)) return false;
	const province = Number(id.slice(0, 2));
	if ((province < 1 || province > 24) && province !== 30) return false;
	if (Number(id[2]) >= 6) return false;
	let sum = 0;
	for (let i = 0; i < 9; i++) {
		let digit = Number(id[i]);
		if (i % 2 === 0) {
			digit *= 2;
			if (digit > 9) digit -= 9;
		}
		sum += digit;
	}
	return (10 - (sum % 10)) % 10 === Number(id[9]);
}

/** A passport number: 5–20 letters or digits (uppercased). */
export const isValidPassport = (value: string) =>
	/^[A-Z0-9]{5,20}$/.test(value);

/** The document as the API stores it: no spaces, a passport uppercased. */
export const normalizeDocument = (type: DocumentType, value: string) => {
	const compact = value.replace(/[\s-]/g, "");
	return type === "passport" ? compact.toUpperCase() : compact;
};

export const isValidDocument = (type: DocumentType, value: string) => {
	const normalized = normalizeDocument(type, value);
	return type === "cedula"
		? isValidCedula(normalized)
		: isValidPassport(normalized);
};

// ─── Birth date ──────────────────────────────────────────────────────────────

export const MAX_AGE_YEARS = 120;

/** The oldest and newest birth dates accepted (YYYY-MM-DD), relative to `today`. */
export function birthDateBounds(today = new Date()) {
	const min = new Date(today);
	min.setFullYear(min.getFullYear() - MAX_AGE_YEARS);
	const max = new Date(today);
	max.setDate(max.getDate() - 1);
	return { min: toDateOnlyString(min), max: toDateOnlyString(max) };
}

/** A real day in the past, at most 120 years ago. */
export function isValidBirthDate(value: string, today = new Date()): boolean {
	if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
	const parsed = parseDateOnly(value);
	// Reject days that roll over (e.g. 2026-02-31).
	if (!parsed || toDateOnlyString(parsed) !== value) return false;
	const { min, max } = birthDateBounds(today);
	return value >= min && value <= max;
}

// ─── Contact ─────────────────────────────────────────────────────────────────

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export const isValidEmail = (value: string) => EMAIL.test(value.trim());

/** Loose: the API normalizes the number to E.164 (Ecuador) and is the judge. */
export const isValidPhone = (value: string) =>
	/^\+?\d{9,15}$/.test(value.replace(/[\s()-]/g, ""));

export const isValidContact = (channel: ContactChannel, value: string) =>
	channel === "email" ? isValidEmail(value) : isValidPhone(value);

/** The channels a new patient may verify with: WhatsApp first when the clinic has it. */
export const contactChannels = (channels: {
	whatsapp: boolean;
	email: boolean;
}): ContactChannel[] => (channels.whatsapp ? ["whatsapp", "email"] : ["email"]);

// ─── Personal data form ──────────────────────────────────────────────────────

export const NAME_MAX = 100;

const name = (key: string) =>
	z
		.string()
		.trim()
		.min(1, key)
		.max(NAME_MAX, "portal.new_patient.error.name_too_long");

export const newPatientSchema = z
	.object({
		first_name: name("portal.new_patient.error.first_name"),
		last_name: name("portal.new_patient.error.last_name"),
		document_type: z.enum(["cedula", "passport"]),
		document: z.string(),
		birth_date: z.string(),
		phone: z.string(),
		email: z.string(),
		whatsapp_opt_in: z.boolean(),
	})
	.superRefine((data, ctx) => {
		if (!isValidDocument(data.document_type, data.document)) {
			ctx.addIssue({
				code: "custom",
				path: ["document"],
				message:
					data.document_type === "cedula"
						? "portal.new_patient.error.cedula"
						: "portal.new_patient.error.passport",
			});
		}
		if (!isValidBirthDate(data.birth_date)) {
			ctx.addIssue({
				code: "custom",
				path: ["birth_date"],
				message: "portal.new_patient.error.birth_date",
			});
		}
		if (data.phone.trim() && !isValidPhone(data.phone)) {
			ctx.addIssue({
				code: "custom",
				path: ["phone"],
				message: "portal.new_patient.error.phone",
			});
		}
		if (data.email.trim() && !isValidEmail(data.email)) {
			ctx.addIssue({
				code: "custom",
				path: ["email"],
				message: "portal.new_patient.error.email",
			});
		}
	});

export type NewPatientValues = z.infer<typeof newPatientSchema>;

/** The contact the person proved they own (prefilled and locked in the form). */
export interface VerifiedContact {
	channel: ContactChannel;
	contact: string;
}

export function initialPatientValues(
	verified: VerifiedContact,
): NewPatientValues {
	return {
		first_name: "",
		last_name: "",
		document_type: "cedula",
		document: "",
		birth_date: "",
		phone: verified.channel === "whatsapp" ? verified.contact : "",
		email: verified.channel === "email" ? verified.contact : "",
		whatsapp_opt_in: false,
	};
}

/** The request's `patient`: trimmed, empty contacts left out, consent only with a phone. */
export function toPatientData(
	values: NewPatientValues,
	verified: VerifiedContact,
): NewPatientData {
	// The verified contact is sent as it was verified, whatever the field shows.
	const phone =
		verified.channel === "whatsapp" ? verified.contact : values.phone.trim();
	const email =
		verified.channel === "email" ? verified.contact : values.email.trim();
	return {
		first_name: values.first_name.trim(),
		last_name: values.last_name.trim(),
		document_type: values.document_type,
		document: normalizeDocument(values.document_type, values.document),
		birth_date: values.birth_date,
		...(phone ? { phone, whatsapp_opt_in: values.whatsapp_opt_in } : {}),
		...(email ? { email } : {}),
	};
}

// ─── Steps ───────────────────────────────────────────────────────────────────

/** After verifying: the booking steps up to the time, then data and review. */
export type NewPatientStep = Exclude<Step, "review"> | "details" | "review";

/** The steps counted in "step X of N" once the contact is verified. */
export function newPatientSteps(bookingSteps: Step[]): NewPatientStep[] {
	return [
		...bookingSteps.filter((s): s is Exclude<Step, "review"> => s !== "review"),
		"details",
		"review",
	];
}

// ─── Errors ──────────────────────────────────────────────────────────────────

export type ContactErrorKind =
	| "invalid_contact"
	| "captcha"
	| "not_accepted"
	| "other";

/** Why asking for a code failed. */
export function contactErrorKind(res: ErrorResponse): ContactErrorKind {
	switch (res.data?.error_code) {
		case NEW_PATIENT_ERROR.invalidContact:
			return "invalid_contact";
		case NEW_PATIENT_ERROR.captcha:
			return "captcha";
		case NEW_PATIENT_ERROR.notAccepted:
			return "not_accepted";
		default:
			return res.code === 404 ? "not_accepted" : "other";
	}
}

export type NewPatientErrorKind =
	| "slot_taken"
	| "already_pending"
	| "clinic_full"
	| "invalid_data"
	| "expired"
	| "other";

/** Why sending the request failed, from its error code or HTTP status. */
export function newPatientErrorKind(res: ErrorResponse): NewPatientErrorKind {
	switch (res.data?.error_code) {
		case BOOKING_ERROR.slotTaken:
			return "slot_taken";
		case NEW_PATIENT_ERROR.alreadyPending:
			return "already_pending";
		case NEW_PATIENT_ERROR.clinicFull:
			return "clinic_full";
		case NEW_PATIENT_ERROR.invalidData:
			return "invalid_data";
	}
	if (res.code === 401) return "expired";
	if (res.code === 429) return "clinic_full";
	if (res.code === 409) return "slot_taken";
	return "other";
}
