import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { api } from ".";

const httpService = createHttpService(api);

export interface Permission {
	ID: string;
	name: string;
	category: string;
	description: string;
}

export const getPermissions = (): Promise<ServiceResponse<Permission[]>> =>
	httpService.get<Permission[]>("/backoffice/permissions");
