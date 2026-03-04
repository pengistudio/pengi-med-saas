import { api } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(api);

export interface BackofficeUser {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	name: string;
	user_name: string;
}

export interface CreateUserRequest extends Record<string, unknown> {
	name: string;
	user_name: string;
	password: string;
}

export interface UpdateUserRequest extends Record<string, unknown> {
	name?: string;
	user_name?: string;
	password?: string;
}

export const getUsers = async (): Promise<ServiceResponse<BackofficeUser[]>> =>
	httpService.get<BackofficeUser[]>("/backoffice/users");

export const getUserByID = async (
	id: number | string,
): Promise<ServiceResponse<BackofficeUser>> =>
	httpService.get<BackofficeUser>(`/backoffice/users/${id}`);

export const createUser = async (
	data: CreateUserRequest,
): Promise<ServiceResponse<BackofficeUser>> =>
	httpService.post<BackofficeUser>("/backoffice/users", data, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateUser = async (
	id: number | string,
	data: UpdateUserRequest,
): Promise<ServiceResponse<BackofficeUser>> =>
	httpService.put<BackofficeUser>(`/backoffice/users/${id}`, data, {
		notifySuccess: true,
		notifyError: true,
	});

export const deleteUser = async (
	id: number | string,
): Promise<ServiceResponse<null>> =>
	httpService.delete<null>(`/backoffice/users/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});
