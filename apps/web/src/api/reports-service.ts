import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from ".";
import type { PaymentMethod } from "./cash-service";

const reportsService = createHttpService(apiWithTenant);

/** Longest range the backend accepts (days, both ends inclusive). */
export const MAX_RANGE_DAYS = 366;

export const REPORT_KINDS = [
	"revenue",
	"receivables",
	"appointments",
	"clinical",
] as const;
export type ReportKind = (typeof REPORT_KINDS)[number];

/** Common filters: dates are YYYY-MM-DD (inclusive); no doctor = all. */
export interface ReportFilters {
	from: string;
	to: string;
	doctor_id?: number;
}

// ── Revenue ─────────────────────────────────────────────────────────────────

export interface ReportEcho {
	from: string;
	to: string;
	doctor_id: number | null;
}

export interface RevenueTotals {
	invoiced: number;
	collected: number;
	receivable: number;
	invoice_count: number;
	payment_count: number;
}

export interface RevenuePoint {
	/** YYYY-MM-DD, or YYYY-MM when the series is monthly. */
	period: string;
	invoiced: number;
	collected: number;
}

export interface RevenueByMethod {
	method: PaymentMethod;
	total: number;
	count: number;
}

export interface RevenueByDoctor {
	doctor_id: number;
	/** "" when the invoice has no doctor. */
	doctor_name: string;
	invoiced: number;
	collected: number;
}

export interface RevenueReport {
	filters: ReportEcho;
	totals: RevenueTotals;
	/** "day", or "month" when the range is longer than 90 days. */
	granularity: "day" | "month";
	series: RevenuePoint[];
	by_method: RevenueByMethod[];
	by_doctor: RevenueByDoctor[];
}

// ── Receivables ─────────────────────────────────────────────────────────────

export interface ReceivableRow {
	invoice_id: number;
	invoice_number: string;
	patient_id: number;
	patient_name: string;
	issue_date: string;
	total: number;
	paid: number;
	balance: number;
	age_days: number;
	payment_status: "unpaid" | "partial";
}

export interface ReceivableBucket {
	key: "0_30" | "31_60" | "61_plus";
	count: number;
	balance: number;
}

export interface ReceivablesReport {
	filters: ReportEcho;
	totals: { count: number; total: number; paid: number; balance: number };
	buckets: ReceivableBucket[];
	items: ReceivableRow[];
}

// ── Appointments ────────────────────────────────────────────────────────────

export interface AppointmentsReport {
	filters: ReportEcho;
	totals: {
		total: number;
		completed: number;
		cancelled: number;
		no_show: number;
		/** 0..1 */
		no_show_rate: number;
	};
	patients: { new: number; recurring: number };
	by_status: { status: string; count: number }[];
	by_doctor: {
		doctor_id: number;
		doctor_name: string;
		total: number;
		completed: number;
		cancelled: number;
		no_show: number;
	}[];
	by_type: { type_id: number; type_name: string; count: number }[];
	series: { day: string; total: number; no_show: number }[];
}

// ── Clinical ────────────────────────────────────────────────────────────────

export interface TopDiagnosis {
	code: string;
	title: string;
	count: number;
}

export interface ClinicalReport {
	filters: ReportEcho;
	totals: { consultations: number; prescriptions: number; exam_orders: number };
	top_diagnoses: TopDiagnosis[];
	by_doctor: {
		doctor_id: number;
		doctor_name: string;
		consultations: number;
		prescriptions: number;
		exam_orders: number;
	}[];
}

// ── Calls ───────────────────────────────────────────────────────────────────

const query = (f: ReportFilters, format?: "csv" | "pdf") => {
	const params = new URLSearchParams({ from: f.from, to: f.to });
	if (f.doctor_id) params.set("doctor_id", String(f.doctor_id));
	if (format) params.set("format", format);
	return `?${params.toString()}`;
};

export const getRevenueReport = (
	f: ReportFilters,
): Promise<ServiceResponse<RevenueReport>> =>
	reportsService.get<RevenueReport>(`/reports/revenue${query(f)}`, {
		notifyError: true,
	});

export const getReceivablesReport = (
	f: ReportFilters,
): Promise<ServiceResponse<ReceivablesReport>> =>
	reportsService.get<ReceivablesReport>(`/reports/receivables${query(f)}`, {
		notifyError: true,
	});

export const getAppointmentsReport = (
	f: ReportFilters,
): Promise<ServiceResponse<AppointmentsReport>> =>
	reportsService.get<AppointmentsReport>(`/reports/appointments${query(f)}`, {
		notifyError: true,
	});

export const getClinicalReport = (
	f: ReportFilters,
): Promise<ServiceResponse<ClinicalReport>> =>
	reportsService.get<ClinicalReport>(`/reports/clinical${query(f)}`, {
		notifyError: true,
	});

/** A report as a CSV or PDF file. */
export const downloadReport = (
	kind: ReportKind,
	format: "csv" | "pdf",
	f: ReportFilters,
): Promise<ServiceResponse<Blob>> =>
	reportsService.get<Blob>(`/reports/${kind}${query(f, format)}`, {
		responseType: "blob",
		notifyError: true,
	});
