import { apiWithTenant } from ".";
import type { Patient } from "./clinical-service";
import {
	type BaseModel,
	createHttpService,
	type ServiceResponse,
} from "./fetch";

const billingService = createHttpService(apiWithTenant);

export interface CatalogService extends BaseModel {
	name: string;
	code: string;
	unit_price: number;
	tax: string;
}

export interface InvoiceItem {
	product_id: number;
	description: string;
	quantity: number;
	unit_price: number;
	discount: number;
	tax_rate: string;
	ice_tax: number;
	total: number;
	product?: CatalogService;
}

export interface Invoice extends BaseModel {
	tenant_id: number;
	patient_id: number;
	patient?: Patient;
	establishment_code: string;
	emission_point_code: string;
	sequential: string;
	authorization_date?: string;
	access_key?: string;
	status: string;
	subtotal: number;
	subtotal_0: number;
	subtotal_12: number;
	total_discount: number;
	total_ice: number;
	total_tax: number;
	total: number;
	items: InvoiceItem[];
}

export type CreateInvoicePayload = {
	patient_id: number;
	items: {
		product_id: number;
		description: string;
		quantity: number;
		unit_price: number;
		discount: number;
		tax_rate: string;
		ice_tax: number;
		total: number;
	}[];
	establishment_code?: string;
	emission_point_code?: string;
};

export const getAllInvoices = async (): Promise<ServiceResponse<Invoice[]>> => {
	return billingService.get<Invoice[]>("/billing/invoices");
};

export const createInvoice = async (
	payload: CreateInvoicePayload,
): Promise<ServiceResponse<Invoice>> => {
	return billingService.post<Invoice>("/billing/invoices", payload);
};

export const deleteInvoice = async (
	id: number,
): Promise<ServiceResponse<null>> => {
	return billingService.delete<null>(`/billing/invoices/${id}`, {
		notifySuccess: true,
	});
};

export const processInvoiceSRI = async (
	id: number,
): Promise<ServiceResponse<null>> => {
	return billingService.post<null>(`/billing/sri/process/${id}`, undefined, {
		notifySuccess: true,
	});
};

export const processMultipleInvoicesSRI = async (
	ids: number[],
): Promise<ServiceResponse<null>> => {
	return billingService.post<null>(
		`/billing/sri/process`,
		{ invoice_ids: ids },
		{ notifySuccess: true },
	);
};
