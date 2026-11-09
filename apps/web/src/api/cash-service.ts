import {
	type BaseModel,
	createHttpService,
	type ServiceResponse,
} from "@pengi/shared";
import { apiWithTenant } from ".";
import type { PaginatedResponse } from "./clinical-service";

const cashService = createHttpService(apiWithTenant);

/** Payment methods, in display order (cash_models.Methods). */
export const PAYMENT_METHODS = [
	"cash",
	"transfer",
	"credit_card",
	"debit_card",
	"check",
	"other",
] as const;
export type PaymentMethod = (typeof PAYMENT_METHODS)[number];

export type MovementType = "in" | "out";
export type MovementCategory =
	| "expense"
	| "withdrawal"
	| "deposit"
	| "adjustment";

/** Categories each movement type allows (movement-handler.go movementTypes). */
export const MOVEMENT_CATEGORIES: Record<MovementType, MovementCategory[]> = {
	in: ["deposit", "adjustment"],
	out: ["expense", "withdrawal", "adjustment"],
};

/** The backend error code of a close whose count differs and has no note. */
export const CASH_NOTE_REQUIRED = "E-CASH-005";

export type InvoicePaymentStatus = "unpaid" | "partial" | "paid";

export interface CashSession extends BaseModel {
	tenant_id: number;
	status: "open" | "closed";
	opened_by: number;
	opened_by_name: string;
	opened_at: string;
	opening_float: number;
	closed_by: number | null;
	closed_by_name: string;
	closed_at: string | null;
	counted_cash: number | null;
	expected_cash: number | null;
	difference: number | null;
	close_note: string;
}

export interface Payment extends BaseModel {
	tenant_id: number;
	invoice_id: number;
	cash_session_id: number;
	method: PaymentMethod;
	/** Negative for a refund. */
	amount: number;
	reference: string;
	kind: "payment" | "refund";
	status: "valid" | "voided";
	reason: string;
	void_reason: string;
	voided_by: number | null;
	voided_at: string | null;
	refund_of_payment_id: number | null;
	created_by: number;
}

/** A payment with the number of the invoice it was made on. */
export interface PaymentView extends Payment {
	invoice_number: string;
}

export interface CashMovement extends BaseModel {
	tenant_id: number;
	cash_session_id: number;
	type: MovementType;
	category: MovementCategory;
	/** Always positive; `type` gives the sign. */
	amount: number;
	reason: string;
	created_by: number;
}

export interface MethodTotal {
	method: PaymentMethod;
	total: number;
	count: number;
}

export interface SessionTotals {
	by_method: MethodTotal[];
	cash_in: number;
	cash_refunded: number;
	movements_in: number;
	movements_out: number;
}

/**
 * A session with its payments, movements and totals. expected_cash and
 * difference stay null while it is open: the UI never shows the expected cash
 * (nor the totals) before the blind close.
 */
export interface CashSessionDetail extends CashSession {
	totals: SessionTotals;
	payments: PaymentView[];
	movements: CashMovement[];
	/** Open for more than 24 hours. */
	stale: boolean;
}

export interface InvoicePayments {
	invoice_id: number;
	total: number;
	paid_amount: number;
	balance: number;
	payment_status: InvoicePaymentStatus;
	payments: PaymentView[];
}

export interface PaymentLinePayload {
	method: PaymentMethod;
	amount: number;
	reference?: string;
}

// Types, not interfaces: the http client takes a Record<string, unknown>.
export type CreateMovementPayload = {
	type: MovementType;
	category: MovementCategory;
	amount: number;
	reason: string;
};

export type RefundPaymentPayload = {
	amount: number;
	reason: string;
	method?: PaymentMethod;
};

// ── Sessions ────────────────────────────────────────────────────────────────

/** The open session, or null when the cash desk is closed. */
export const getCurrentSession = async (): Promise<
	ServiceResponse<CashSessionDetail | null>
> =>
	cashService.get<CashSessionDetail | null>("/cash/sessions/current", {
		notifyError: true,
	});

export type SessionListParams = {
	page?: number;
	limit?: number;
	status?: "open" | "closed";
};

export const getSessions = async (
	params: SessionListParams = {},
): Promise<ServiceResponse<PaginatedResponse<CashSession>>> => {
	const qs = new URLSearchParams();
	if (params.page) qs.set("page", String(params.page));
	if (params.limit) qs.set("limit", String(params.limit));
	if (params.status) qs.set("status", params.status);
	const query = qs.toString() ? `?${qs.toString()}` : "";
	return cashService.get<PaginatedResponse<CashSession>>(
		`/cash/sessions${query}`,
		{ notifyError: true },
	);
};

export const getSession = async (
	id: number,
): Promise<ServiceResponse<CashSessionDetail>> =>
	cashService.get<CashSessionDetail>(`/cash/sessions/${id}`, {
		notifyError: true,
	});

export const openSession = async (
	openingFloat: number,
): Promise<ServiceResponse<CashSessionDetail>> =>
	cashService.post<CashSessionDetail>(
		"/cash/sessions",
		{ opening_float: openingFloat },
		{ notifySuccess: true, notifyError: true },
	);

/**
 * Blind close: sends only the counted cash. A count that differs without a
 * note fails with CASH_NOTE_REQUIRED and reveals no amount.
 */
export const closeSession = async (
	id: number,
	payload: { counted_cash: number; note?: string },
): Promise<ServiceResponse<CashSessionDetail>> =>
	cashService.post<CashSessionDetail>(`/cash/sessions/${id}/close`, payload, {
		notifySuccess: true,
		notifyError: true,
	});

/** The closing report (PDF) of a closed session. */
export const downloadSessionReport = async (
	id: number,
): Promise<ServiceResponse<Blob>> =>
	cashService.get<Blob>(`/cash/sessions/${id}/report`, {
		responseType: "blob",
		notifyError: true,
	});

// ── Payments ────────────────────────────────────────────────────────────────

/** What an invoice has been charged: paid amount, balance and payments. */
export const getInvoicePayments = async (
	invoiceId: number,
): Promise<ServiceResponse<InvoicePayments>> =>
	cashService.get<InvoicePayments>(`/cash/payments?invoice_id=${invoiceId}`, {
		notifyError: true,
	});

/** Charges an invoice in the open session; all lines are saved or none. */
export const createPayment = async (
	invoiceId: number,
	lines: PaymentLinePayload[],
): Promise<ServiceResponse<InvoicePayments>> =>
	cashService.post<InvoicePayments>(
		"/cash/payments",
		{ invoice_id: invoiceId, lines },
		{ notifySuccess: true, notifyError: true },
	);

/** Voids a payment of the open session. */
export const voidPayment = async (
	id: number,
	reason: string,
): Promise<ServiceResponse<Payment>> =>
	cashService.post<Payment>(
		`/cash/payments/${id}/void`,
		{ reason },
		{ notifySuccess: true, notifyError: true },
	);

/** Gives back part or all of a payment of a closed session. */
export const refundPayment = async (
	id: number,
	payload: RefundPaymentPayload,
): Promise<ServiceResponse<Payment>> =>
	cashService.post<Payment>(`/cash/payments/${id}/refund`, payload, {
		notifySuccess: true,
		notifyError: true,
	});

// ── Movements ───────────────────────────────────────────────────────────────

export const createMovement = async (
	payload: CreateMovementPayload,
): Promise<ServiceResponse<CashMovement>> =>
	cashService.post<CashMovement>("/cash/movements", payload, {
		notifySuccess: true,
		notifyError: true,
	});
