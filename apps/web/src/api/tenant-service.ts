import { apiWithTenant } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const tenantService = createHttpService(apiWithTenant);

export const uploadSriSignature = async (file: File, password: string) => {
	const formData = new FormData();
	formData.append("signature", file);
	formData.append("password", password);

	return tenantService.putForm("/tenants/sri/signature", formData, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const uploadLogo = async (file: File) => {
	const formData = new FormData();
	formData.append("logo", file);

	return tenantService.putForm("/tenants/logo", formData, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const getLogo = async (): Promise<ServiceResponse<Blob>> => {
	return tenantService.get<Blob>("/tenants/logo", {
		responseType: "blob",
		notifyError: false,
	});
};

export type SriStatus = {
	is_configured: boolean;
	expiration_date: string | null;
	tax_id: string;
	trade_name: string;
	corporate_name: string;
	address: string;
	accounting_obliged: boolean;
	special_contributor_number: string;
	microenterprise_regime: boolean;
	withholding_agent: string;
	rimpe_taxpayer: string;
	has_logo: boolean;
};

export const getSriStatus = async () => {
	return tenantService.get<SriStatus>("/tenants/sri/status", {
		notifyError: true,
	});
};

export type UpdateSriInfoPayload = {
	tax_id: string;
	trade_name: string;
	corporate_name: string;
	address: string;
	accounting_obliged: boolean;
	special_contributor_number: string;
	microenterprise_regime: boolean;
	withholding_agent: string;
	rimpe_taxpayer: string;
};

export const updateSriInfo = async (payload: UpdateSriInfoPayload) => {
	return tenantService.put<SriStatus>("/tenants/sri/info", payload, {
		notifySuccess: true,
		notifyError: true,
	});
};
