import { createHttpService } from "@pengi/shared";
import { portalApi } from "./http-clients";

const portalService = createHttpService(portalApi);

export interface PortalSections {
	appointments: boolean;
	results: boolean;
	documents: boolean;
	invoices: boolean;
}

export interface Clinic {
	name: string;
	slug: string;
	has_logo: boolean;
	sections: PortalSections;
	channels: { whatsapp: boolean; email: boolean };
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
	clinic: Pick<Clinic, "name" | "slug" | "has_logo" | "sections">;
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
