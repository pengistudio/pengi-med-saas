import { apiWithTenant } from ".";
import { createHttpService } from "./fetch";

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

export type SriStatus = {
	is_configured: boolean;
	expiration_date: string | null;
	tax_id: string;
	trade_name: string;
	corporate_name: string;
	address: string;
	accounting_obliged: boolean;
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
};

export const updateSriInfo = async (payload: UpdateSriInfoPayload) => {
	return tenantService.put<SriStatus>("/tenants/sri/info", payload, {
		notifySuccess: true,
		notifyError: true,
	});
};
