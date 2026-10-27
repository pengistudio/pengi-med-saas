import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from ".";

const appointmentRequestService = createHttpService(apiWithTenant);

/** A patient's request for an appointment from the portal. */
export type AppointmentRequestStatus =
	| "pending"
	| "approved"
	| "rejected"
	| "cancelled"
	| "expired";

export type AppointmentRequestFilter = "pending" | "all";

export interface AppointmentRequest {
	id: number;
	status: AppointmentRequestStatus;
	patient: {
		id: number;
		first_name: string;
		last_name: string;
		document: string;
		phone: string;
	};
	doctor: {
		id: number;
		full_name: string;
		specialty: string;
		specialty_other: string;
	};
	type_name: string;
	/** YYYY-MM-DD and "HH:MM". */
	date: string;
	start_time: string;
	end_time: string;
	/** The patient's note; may be empty. */
	reason: string;
	reject_reason?: string;
	/** ISO timestamps. */
	expires_at: string;
	created_at: string;
	/** Set once approved. */
	appointment_id?: number | null;
}

export const getAppointmentRequests = async (
	status: AppointmentRequestFilter,
): Promise<ServiceResponse<AppointmentRequest[]>> =>
	appointmentRequestService.get<AppointmentRequest[]>("/appointment-requests", {
		params: { status },
		notifyError: true,
	});

/** For the nav badge: silent on errors (the user may lack access). */
export const getPendingAppointmentRequestCount = async (): Promise<number> => {
	const res = await appointmentRequestService.get<{ count: number }>(
		"/appointment-requests/pending-count",
	);
	return res.success ? (res.data?.count ?? 0) : 0;
};

export const approveAppointmentRequest = async (
	id: number,
): Promise<ServiceResponse<AppointmentRequest>> =>
	appointmentRequestService.post<AppointmentRequest>(
		`/appointment-requests/${id}/approve`,
		undefined,
		{ notifySuccess: true, notifyError: true },
	);

export const rejectAppointmentRequest = async (
	id: number,
	reason: string,
): Promise<ServiceResponse<AppointmentRequest>> =>
	appointmentRequestService.post<AppointmentRequest>(
		`/appointment-requests/${id}/reject`,
		reason ? { reason } : {},
		{ notifySuccess: true, notifyError: true },
	);
