import { apiWithTenant } from ".";
import type { Patient } from "./clinical-service";
import type { PaginatedResponse } from "./clinical-service";
import {
	type BaseModel,
	createHttpService,
	type ServiceResponse,
} from "./fetch";

const billingService = createHttpService(apiWithTenant);

export interface CatalogItem extends BaseModel {
	name: string;
	sku: string;
	description: string;
	unit_price: number;
	tax: number;
	tax_code: string;
	tax_percentage_code: string;
	ice_tax: number;
	ice_tax_code: string;
	ice_tax_percentage_code: string;
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
	product?: CatalogItem;
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

export type CreateCatalogItemPayload = {
	name: string;
	sku: string;
	description?: string;
	unit_price: number;
	tax?: number;
	tax_code?: string;
	tax_percentage_code?: string;
	ice_tax?: number;
	ice_tax_code?: string;
	ice_tax_percentage_code?: string;
};

export type UpdateCatalogItemPayload = Partial<CreateCatalogItemPayload>;

export type InvoiceListParams = {
	page?: number;
	limit?: number;
	search?: string;
};

export const getAllInvoices = async (
	params: InvoiceListParams = {},
): Promise<ServiceResponse<PaginatedResponse<Invoice>>> => {
	const qs = new URLSearchParams();
	if (params.page) qs.set("page", String(params.page));
	if (params.limit) qs.set("limit", String(params.limit));
	if (params.search) qs.set("search", params.search);
	const query = qs.toString() ? `?${qs.toString()}` : "";
	return billingService.get<PaginatedResponse<Invoice>>(
		`/billing/invoices${query}`,
		{ notifyError: true },
	);
};

export const createInvoice = async (
	payload: CreateInvoicePayload,
): Promise<ServiceResponse<Invoice>> => {
	return billingService.post<Invoice>("/billing/invoices", payload, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const deleteInvoice = async (
	id: number,
): Promise<ServiceResponse<null>> => {
	return billingService.delete<null>(`/billing/invoices/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const processInvoiceSRI = async (
	id: number,
): Promise<ServiceResponse<null>> => {
	return billingService.post<null>(
		`/billing/invoices/${id}/sri/process`,
		undefined,
		{
			notifySuccess: true,
			notifyError: true,
		},
	);
};

export const processMultipleInvoicesSRI = async (
	ids: number[],
): Promise<ServiceResponse<null>> => {
	return billingService.post<null>(
		`/billing/invoices/sri/process-batch`,
		{ id_list: ids },
		{
			notifySuccess: true,
			notifyError: true,
		},
	);
};

export type CatalogItemListParams = {
	page?: number;
	limit?: number;
	search?: string;
};

export const getAllCatalogItems = async (
	params: CatalogItemListParams = {},
): Promise<ServiceResponse<PaginatedResponse<CatalogItem>>> => {
	const qs = new URLSearchParams();
	if (params.page) qs.set("page", String(params.page));
	if (params.limit) qs.set("limit", String(params.limit));
	if (params.search) qs.set("search", params.search);
	const query = qs.toString() ? `?${qs.toString()}` : "";
	return billingService.get<PaginatedResponse<CatalogItem>>(
		`/billing/catalog-items${query}`,
		{ notifyError: true },
	);
};

export const getCatalogItemById = async (
	id: number,
): Promise<ServiceResponse<CatalogItem>> => {
	return billingService.get<CatalogItem>(`/billing/catalog-items/${id}`, {
		notifyError: true,
	});
};

export const createCatalogItem = async (
	payload: CreateCatalogItemPayload,
): Promise<ServiceResponse<CatalogItem>> => {
	return billingService.post<CatalogItem>("/billing/catalog-items", payload, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const updateCatalogItem = async (
	id: number,
	payload: UpdateCatalogItemPayload,
): Promise<ServiceResponse<CatalogItem>> => {
	return billingService.put<CatalogItem>(
		`/billing/catalog-items/${id}`,
		payload,
		{
			notifySuccess: true,
			notifyError: true,
		},
	);
};

export const deleteCatalogItem = async (
	id: number,
): Promise<ServiceResponse<null>> => {
	return billingService.delete<null>(`/billing/catalog-items/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});
};

