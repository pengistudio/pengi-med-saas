import { api } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(api);

export interface Company {
	ID: number;
	createdAt: string;
	updatedAt: string;
	legal_name: string;
	trade_name: string;
	plan_code: string;
	tenant_id: number;
	tenant: {
		ID: number;
		name: string;
		slug: string;
	};
}

export interface CreateCompanyRequest extends Record<string, unknown> {
	legal_name: string;
	trade_name: string;
	plan_code: string;
}

export interface UpdateCompanyRequest extends Record<string, unknown> {
	legal_name?: string;
	trade_name?: string;
	plan_code?: string;
}

export const getCompanies = async (): Promise<ServiceResponse<Company[]>> => {
	return httpService.get<Company[]>("/backoffice/companies");
};

export const getCompanyByID = async (
	id: number | string,
): Promise<ServiceResponse<Company>> => {
	return httpService.get<Company>(`/backoffice/companies/${id}`);
};

export const createCompany = async (
	data: CreateCompanyRequest,
): Promise<ServiceResponse<Company>> => {
	return httpService.post<Company>("/backoffice/companies", data, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const updateCompany = async (
	id: number | string,
	data: UpdateCompanyRequest,
): Promise<ServiceResponse<Company>> => {
	return httpService.put<Company>(`/backoffice/companies/${id}`, data, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const deleteCompany = async (
	id: number | string,
): Promise<ServiceResponse<null>> => {
	return httpService.delete<null>(`/backoffice/companies/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});
};
