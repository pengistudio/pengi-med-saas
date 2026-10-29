import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { guestConfig, portalApi } from "./http-clients";

const portalService = createHttpService(portalApi);

export interface PortalSections {
	appointments: boolean;
	results: boolean;
	documents: boolean;
	invoices: boolean;
}

/** How patients pick who sees them: a named doctor, or any doctor of a specialty. */
export type BookingChooseBy = "doctor" | "specialty";

/** Booking from the portal; on only when the clinic turned it on. */
export interface ClinicBooking {
	enabled: boolean;
	choose_by: BookingChooseBy;
	/** People without a patient record may request a first appointment. */
	new_patients?: boolean;
}

export interface Clinic {
	name: string;
	slug: string;
	has_logo: boolean;
	sections: PortalSections;
	channels: { whatsapp: boolean; email: boolean };
	/** Absent on an API without booking: treat as off. */
	booking?: ClinicBooking;
}

export interface PortalPatient {
	first_name: string;
	last_name: string;
	document: string;
	email: string;
	phone: string;
}

export interface Me {
	patient: PortalPatient;
	clinic: Pick<Clinic, "name" | "slug" | "has_logo" | "sections" | "booking">;
}

export interface PortalAppointment {
	id: number;
	date: string; // YYYY-MM-DD
	start_time: string;
	end_time: string;
	status: string;
	doctor_name: string;
	/** Specialty code; `specialty_other` holds the name when it is "other". */
	specialty: string;
	specialty_other: string;
	appointment_type: string;
	location: string;
	/** The patient may still cancel it from the portal (clinic's notice window). */
	can_cancel?: boolean;
}

export interface ResultFile {
	id: number;
	file_name: string;
	content_type: string;
}

export interface ResultItem {
	id: number;
	name: string;
	reviewed_at: string;
	files: ResultFile[];
}

export interface ResultOrder {
	id: number;
	number: string;
	date: string;
	doctor_name: string;
	items: ResultItem[];
}

export type PortalDocumentType = "prescription" | "certificate" | "report";

export interface PortalDocument {
	type: PortalDocumentType;
	id: number;
	date: string;
	doctor_name: string;
	signed_at: string;
}

export interface PortalInvoice {
	id: number;
	number: string;
	date: string;
	total: number;
	access_key: string;
}

/** Whether the clinic has the portal on; the slug is public. */
export const getClinic = (slug: string) =>
	portalService.get<Clinic>(`/clinics/${encodeURIComponent(slug)}`);

// A type, not an interface: the HTTP client takes a Record<string, unknown>.
export type LoginRequest = {
	document: string;
	/** Honeypot: empty for people. */
	website: string;
	/** Cloudflare Turnstile token, required when the API has a Turnstile secret. */
	captcha_token?: string;
};

/**
 * Same answer whether or not the document exists. A missing or rejected
 * captcha answers 400 E-PORTAL-008.
 */
export const requestCode = (slug: string, request: LoginRequest) =>
	portalService.post<{ expires_in: number }>(
		`/clinics/${encodeURIComponent(slug)}/login`,
		request,
		{ notifyError: true },
	);

/** Errors are shown inline by the caller, not as a toast. */
export const verifyCode = (slug: string, document: string, code: string) =>
	portalService.post<{ access_token: string }>(
		`/clinics/${encodeURIComponent(slug)}/verify`,
		{ document, code },
	);

export const logout = () => portalService.post<null>("/logout");

export const getMe = () => portalService.get<Me>("/me");

export const getAppointments = () =>
	portalService.get<PortalAppointment[]>("/appointments");

export const getResults = () => portalService.get<ResultOrder[]>("/results");

export const getDocuments = () =>
	portalService.get<PortalDocument[]>("/documents");

export const getInvoices = () =>
	portalService.get<PortalInvoice[]>("/invoices");

const download = (url: string) =>
	portalService.get<Blob>(url, { responseType: "blob", notifyError: true });

export const downloadResultFile = (id: number) =>
	download(`/results/files/${id}`);

export const downloadDocument = (type: PortalDocumentType, id: number) =>
	download(`/documents/${type}/${id}/file`);

export const downloadInvoiceRide = (id: number) =>
	download(`/invoices/${id}/ride`);

export const downloadInvoiceXml = (id: number) =>
	download(`/invoices/${id}/xml`);

// ─── Booking (Entrega 2) ─────────────────────────────────────────────────────

/** A doctor patients can book; `type_ids` are the appointment types it offers. */
export interface BookingDoctor {
	id: number;
	full_name: string;
	specialty: string;
	specialty_other: string;
	type_ids: number[];
}

/** A specialty patients can book; `type_ids` are the types its doctors offer. */
export interface BookingSpecialty {
	specialty: string;
	specialty_other: string;
	type_ids: number[];
}

export interface BookingType {
	id: number;
	name: string;
	duration_minutes: number;
}

export interface BookingOptions {
	choose_by: BookingChooseBy;
	/** First and last bookable day, YYYY-MM-DD. */
	min_date: string;
	max_date: string;
	/** May be empty in specialty mode. */
	doctors?: BookingDoctor[];
	specialties: BookingSpecialty[];
	types: BookingType[];
}

export interface SlotDay {
	date: string;
	/** "HH:MM" start times, ascending. */
	times: string[];
}

export interface BookingSlots {
	days: SlotDay[];
}

/** Who the slots are for: one doctor, or a specialty (any of its doctors). */
export type BookingTarget =
	| { doctor_id: number }
	| { specialty: string; specialty_other: string };

export type AppointmentRequestStatus =
	| "pending"
	| "approved"
	| "rejected"
	| "cancelled"
	| "expired";

export interface PortalAppointmentRequest {
	id: number;
	status: AppointmentRequestStatus;
	date: string;
	start_time: string;
	end_time: string;
	doctor: { full_name: string; specialty: string; specialty_other: string };
	type_name: string;
	reason: string;
	reject_reason: string;
	/** ISO timestamp; a pending request lapses then. */
	expires_at: string;
}

// A type, not an interface: the HTTP client takes a Record<string, unknown>.
export type CreateBookingRequest = {
	type_id: number;
	date: string;
	start_time: string;
	reason?: string;
	doctor_id?: number;
	specialty?: string;
	specialty_other?: string;
};

/**
 * Error codes of the booking endpoints. The HTTP status is the fallback when
 * the code is not one of these: 409 = slot taken, 429 (the IP rate limit) is
 * shown like too many pending requests.
 */
export const BOOKING_ERROR = {
	slotTaken: "E-PORTAL-010",
	tooManyPending: "E-PORTAL-011",
	tooLateToCancel: "E-PORTAL-012",
	notPending: "E-PORTAL-013",
} as const;

/** With a guest token, for a new patient (no session). */
export const getBookingOptions = (guestToken?: string) =>
	portalService.get<BookingOptions>(
		"/booking/options",
		guestToken ? guestConfig(guestToken) : undefined,
	);

/** Loads the free slots of a page of days (patient session or guest token). */
export type SlotsLoader = (
	typeId: number,
	target: BookingTarget,
	from: string,
	to: string,
) => Promise<ServiceResponse<BookingSlots>>;

export const getBookingSlots: SlotsLoader = (typeId, target, from, to) =>
	portalService.get<BookingSlots>("/booking/slots", {
		params: { type_id: typeId, ...target, from, to },
	});

/** The slots loader of a new patient's guest token. */
export const guestSlotsLoader =
	(guestToken: string): SlotsLoader =>
	(typeId, target, from, to) =>
		portalService.get<BookingSlots>("/booking/slots", {
			...guestConfig(guestToken),
			params: { type_id: typeId, ...target, from, to },
		});

/** Errors (slot taken, too many pending) are shown inline by the caller. */
export const createBookingRequest = (request: CreateBookingRequest) =>
	portalService.post<PortalAppointmentRequest>("/booking/requests", request);

export const getBookingRequests = () =>
	portalService.get<PortalAppointmentRequest[]>("/booking/requests");

export const cancelBookingRequest = (id: number) =>
	portalService.post<PortalAppointmentRequest>(
		`/booking/requests/${id}/cancel`,
		undefined,
		{ notifySuccess: true, notifyError: true },
	);

export const cancelAppointment = (id: number) =>
	portalService.post<PortalAppointment>(
		`/appointments/${id}/cancel`,
		undefined,
		{ notifySuccess: true, notifyError: true },
	);

// ─── Waitlist ────────────────────────────────────────────────────────────────

/** The patient asked to be told when a slot frees (with a doctor, or any). */
export interface WaitlistEntry {
	id: number;
	status: "waiting" | "offered" | "booked" | "removed" | "expired";
	/** Null: any doctor (specialty mode). */
	doctor_id: number | null;
	doctor_name: string;
	type_id: number | null;
	type_name: string;
	/** ISO timestamps. */
	created_at: string;
	expires_at: string;
}

/** The patient's entries still on the list. */
export const getWaitlist = () =>
	portalService.get<WaitlistEntry[]>("/waitlist");

/** Joins the list for a type (and doctor); joining twice returns the same entry. */
export const joinWaitlist = (typeId: number, doctorId?: number) =>
	portalService.post<WaitlistEntry>(
		"/waitlist",
		{ type_id: typeId, ...(doctorId ? { doctor_id: doctorId } : {}) },
		{ notifyError: true },
	);

export const leaveWaitlist = (id: number) =>
	portalService.delete<WaitlistEntry>(`/waitlist/${id}`, {
		notifyError: true,
	});

// ─── New patients (Entrega 3) ────────────────────────────────────────────────

/** How a new patient proves a contact: WhatsApp only when the clinic has it. */
export type ContactChannel = "whatsapp" | "email";

export type DocumentType = "cedula" | "passport";

// A type, not an interface: the HTTP client takes a Record<string, unknown>.
export type NewPatientCodeRequest = {
	channel: ContactChannel;
	contact: string;
	/** Honeypot: empty for people. */
	website: string;
	captcha_token?: string;
};

/**
 * Same answer for any well-formed contact. Errors are shown inline: 400
 * E-PORTAL-005 (malformed contact or channel off), 400 E-PORTAL-008 (captcha),
 * 404 E-PORTAL-015 (the clinic does not take new patients), 429 (IP limit).
 */
export const requestNewPatientCode = (
	slug: string,
	request: NewPatientCodeRequest,
) =>
	portalService.post<{ expires_in: number }>(
		`/clinics/${encodeURIComponent(slug)}/new-patient/code`,
		request,
	);

/** Every failure answers 401 E-PORTAL-002; shown inline. */
export const verifyNewPatientCode = (
	slug: string,
	request: { channel: ContactChannel; contact: string; code: string },
) =>
	portalService.post<{ guest_token: string; expires_in: number }>(
		`/clinics/${encodeURIComponent(slug)}/new-patient/verify`,
		request,
	);

export type NewPatientData = {
	first_name: string;
	last_name: string;
	document_type: DocumentType;
	document: string;
	/** YYYY-MM-DD. */
	birth_date: string;
	phone?: string;
	email?: string;
	whatsapp_opt_in?: boolean;
};

export type CreateNewPatientRequest = CreateBookingRequest & {
	patient: NewPatientData;
};

export interface NewPatientRequestResult {
	id: number;
	status: AppointmentRequestStatus;
	date: string;
	start_time: string;
	end_time: string;
	doctor: { full_name: string; specialty: string; specialty_other: string };
	type_name: string;
	expires_at: string;
	/** The verified channel the clinic will answer by. */
	contact_channel: ContactChannel;
}

/**
 * Error codes of the new-patient endpoints (shown inline). Slot taken is
 * BOOKING_ERROR.slotTaken, as for patients.
 */
export const NEW_PATIENT_ERROR = {
	invalidContact: "E-PORTAL-005",
	captcha: "E-PORTAL-008",
	notAccepted: "E-PORTAL-015",
	invalidData: "E-PORTAL-016",
	alreadyPending: "E-PORTAL-017",
	clinicFull: "E-PORTAL-018",
} as const;

/** Always pending: the clinic approves every new patient. */
export const createNewPatientRequest = (
	guestToken: string,
	request: CreateNewPatientRequest,
) =>
	portalService.post<NewPatientRequestResult>(
		"/new-patient/requests",
		request,
		guestConfig(guestToken),
	);
