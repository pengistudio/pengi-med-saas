import type { BaseModel } from "@pengi/shared";
import { resource } from "@/lib/resource/http-resource";
export interface BackofficeUser extends BaseModel {
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

export const users = resource<
	BackofficeUser,
	CreateUserRequest,
	UpdateUserRequest
>("users");
