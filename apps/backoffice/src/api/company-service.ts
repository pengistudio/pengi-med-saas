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
	Subscriptions?: {
		ID: number;
		plan_code: string;
		status: string;
		expires_at: string;
	}[];
}

export interface CreateCompanyRequest extends Record<string, unknown> {
	legal_name: string;
	trade_name: string;
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

export interface CompanySignupTokenResponse {
	token: string;
	company_id: number;
	trade_name: string;
}

export const getCompanySignupToken = async (
	id: number | string,
): Promise<ServiceResponse<CompanySignupTokenResponse>> => {
	return httpService.get<CompanySignupTokenResponse>(
		`/backoffice/companies/${id}/signup-token`,
	);
};

// ── Company Users ───────────────────────────────────────────────────────────

export interface CompanyUser {
	environment_id: number;
	user_id: number;
	user_name: string;
	email: string;
	role_id: number;
	role_name: string;
	environment_name: string;
}

export interface Role {
	ID: number;
	role: string;
}

export interface UpdateCompanyUserRequest extends Record<string, unknown> {
	user_name?: string;
	email?: string;
	role_id?: number;
}

export const getCompanyUsers = async (
	companyId: number | string,
): Promise<ServiceResponse<CompanyUser[]>> => {
	return httpService.get<CompanyUser[]>(
		`/backoffice/companies/${companyId}/users`,
	);
};

export const updateCompanyUser = async (
	companyId: number | string,
	userId: number | string,
	data: UpdateCompanyUserRequest,
): Promise<ServiceResponse<null>> => {
	return httpService.put<null>(
		`/backoffice/companies/${companyId}/users/${userId}`,
		data,
		{ notifySuccess: true, notifyError: true },
	);
};

export const getRoles = async (): Promise<ServiceResponse<Role[]>> => {
	return httpService.get<Role[]>("/backoffice/roles");
};

export interface PasswordResetLinkResponse {
	link: string;
	user_id: number;
	username: string;
	email: string;
}

export const getPasswordResetLink = async (
	companyId: number | string,
	userId: number | string,
): Promise<ServiceResponse<PasswordResetLinkResponse>> => {
	return httpService.get<PasswordResetLinkResponse>(
		`/backoffice/companies/${companyId}/users/${userId}/password-reset-link`,
		{ notifyError: true },
	);
};
