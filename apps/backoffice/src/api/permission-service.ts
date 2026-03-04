import { api } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(api);

export interface Permission {
	ID: string;
	name: string;
	category: string;
	description: string;
}

export const getPermissions = (): Promise<ServiceResponse<Permission[]>> =>
	httpService.get<Permission[]>("/backoffice/permissions");
