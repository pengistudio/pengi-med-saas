import { api } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(api);

export interface Role {
	ID: number;
	createdAt: string;
	updatedAt: string;
	role: string;
	permissions: {
		ID: string;
		name: string;
		category: string;
		description: string;
	}[];
}

export interface CreateRoleRequest extends Record<string, unknown> {
	role: string;
	permission_ids?: string[];
}

export interface UpdateRoleRequest extends Record<string, unknown> {
	role?: string;
	permission_ids?: string[];
}

export const getRoles = (): Promise<ServiceResponse<Role[]>> =>
	httpService.get<Role[]>("/backoffice/roles");

export const getRoleByID = (
	id: number | string,
): Promise<ServiceResponse<Role>> =>
	httpService.get<Role>(`/backoffice/roles/${id}`);

export const createRole = (
	data: CreateRoleRequest,
): Promise<ServiceResponse<Role>> =>
	httpService.post<Role>("/backoffice/roles", data, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateRole = (
	id: number | string,
	data: UpdateRoleRequest,
): Promise<ServiceResponse<Role>> =>
	httpService.put<Role>(`/backoffice/roles/${id}`, data, {
		notifySuccess: true,
		notifyError: true,
	});

export const deleteRole = (
	id: number | string,
): Promise<ServiceResponse<null>> =>
	httpService.delete<null>(`/backoffice/roles/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});
