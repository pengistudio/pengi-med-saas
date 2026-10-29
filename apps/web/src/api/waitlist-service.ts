import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from ".";

const waitlistService = createHttpService(apiWithTenant);

/** waiting and offered are on the list; the rest are closed. */
export type WaitlistStatus =
	| "waiting"
	| "offered"
	| "booked"
	| "removed"
	| "expired";

/** The list's status filter: "active" = waiting + offered. */
export type WaitlistStatusFilter = "active" | WaitlistStatus | "all";

export interface WaitlistEntry {
	id: number;
	status: WaitlistStatus;
	/** staff = reception; portal = the patient, from the portal. */
	source: "staff" | "portal";
	patient: {
		id: number;
		first_name: string;
		last_name: string;
		document: string;
		phone: string;
		whatsapp_opt_in: boolean;
	};
	/** Null: any doctor. */
	doctor: { id: number; full_name: string } | null;
	/** Null: any type. */
	appointment_type: {
		id: number;
		name: string;
		duration_minutes: number;
	} | null;
	/** YYYY-MM-DD or "" (open). */
	date_from: string;
	date_to: string;
	/** The appointment the patient wants to move earlier. */
	appointment: {
		id: number;
		date: string;
		start_time: string;
		status: string;
	} | null;
	notes: string;
	/** ISO timestamps. */
	created_at: string;
	expires_at: string;
	/** The last slot offered. */
	offer: {
		doctor_id: number | null;
		date: string;
		start_time: string;
		end_time: string;
		channel: "whatsapp" | "phone";
		offered_at: string | null;
		/** The patient tapped "Lo quiero". */
		accepted_at: string | null;
	} | null;
	/** Candidates only: consent and a valid phone for a WhatsApp offer. */
	can_whatsapp?: boolean;
	/** Candidates only: this slot was already offered to the entry. */
	offered_this_slot?: boolean;
}

/** A freed slot: YYYY-MM-DD and "HH:MM" times. */
export interface WaitlistSlot {
	date: string;
	start_time: string;
	end_time?: string;
	doctor_id?: number | null;
	appointment_type_id?: number | null;
	/** Who freed it: never offered their own slot. */
	patient_id?: number | null;
}

export interface WaitlistCandidates {
	/** The clinic can offer by WhatsApp (plan, connection, approved template). */
	whatsapp_available: boolean;
	/** The slot's doctor name ("" when none). */
	doctor: string;
	candidates: WaitlistEntry[];
}

// A type, not an interface: the HTTP client takes a Record<string, unknown>.
export type WaitlistEntryPayload = {
	patient_id?: number;
	/** 0 / null: any. */
	doctor_id: number | null;
	appointment_type_id: number | null;
	date_from: string;
	date_to: string;
	appointment_id: number | null;
	notes: string;
};

export type OfferChannel = "whatsapp" | "phone";

export const waitlistPatientName = (entry: Pick<WaitlistEntry, "patient">) =>
	`${entry.patient.first_name} ${entry.patient.last_name}`.trim();

/** Reads a freed slot from the waitlist URL (the notification's link). */
export function slotFromSearch(search: URLSearchParams): WaitlistSlot | null {
	const date = search.get("date");
	const start = search.get("start_time");
	if (!date || !start) return null;
	const id = (name: string) => {
		const n = Number(search.get(name));
		return Number.isInteger(n) && n > 0 ? n : null;
	};
	return {
		date,
		start_time: start,
		end_time: search.get("end_time") ?? "",
		doctor_id: id("doctor_id"),
		appointment_type_id: id("appointment_type_id"),
		patient_id: id("patient_id"),
	};
}

const slotParams = (slot: WaitlistSlot) => ({
	date: slot.date,
	start_time: slot.start_time,
	...(slot.end_time ? { end_time: slot.end_time } : {}),
	...(slot.doctor_id ? { doctor_id: slot.doctor_id } : {}),
	...(slot.appointment_type_id
		? { appointment_type_id: slot.appointment_type_id }
		: {}),
	...(slot.patient_id ? { patient_id: slot.patient_id } : {}),
});

export const getWaitlist = async (filters: {
	status: WaitlistStatusFilter;
	/** A doctor's id, "any" (entries for any doctor) or undefined (all). */
	doctor?: number | "any";
}): Promise<ServiceResponse<WaitlistEntry[]>> =>
	waitlistService.get<WaitlistEntry[]>("/agenda/waitlist", {
		params: {
			status: filters.status,
			...(filters.doctor ? { doctor_id: filters.doctor } : {}),
		},
		notifyError: true,
	});

export const getWaitlistCandidates = async (
	slot: WaitlistSlot,
): Promise<ServiceResponse<WaitlistCandidates>> =>
	waitlistService.get<WaitlistCandidates>("/agenda/waitlist/candidates", {
		params: slotParams(slot),
		notifyError: true,
	});

export const createWaitlistEntry = async (
	payload: WaitlistEntryPayload,
): Promise<ServiceResponse<WaitlistEntry>> =>
	waitlistService.post<WaitlistEntry>("/agenda/waitlist", payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateWaitlistEntry = async (
	id: number,
	payload: WaitlistEntryPayload,
): Promise<ServiceResponse<WaitlistEntry>> =>
	waitlistService.put<WaitlistEntry>(`/agenda/waitlist/${id}`, payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const removeWaitlistEntry = async (
	id: number,
): Promise<ServiceResponse<WaitlistEntry>> =>
	waitlistService.delete<WaitlistEntry>(`/agenda/waitlist/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});

export const offerWaitlistSlot = async (
	id: number,
	slot: WaitlistSlot,
	channel: OfferChannel,
): Promise<ServiceResponse<WaitlistEntry>> =>
	waitlistService.post<WaitlistEntry>(
		`/agenda/waitlist/${id}/offer`,
		{ channel, ...slotParams(slot) },
		{ notifySuccess: true, notifyError: true },
	);
