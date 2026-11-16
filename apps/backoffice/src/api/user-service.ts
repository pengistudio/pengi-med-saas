import type { BaseModel } from "@pengi/shared";
import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { api } from "@/api";
import { resource } from "@/lib/resource/http-resource";
export interface BackofficeUser extends BaseModel {
	name: string;
	user_name: string;
	two_factor_enabled: boolean;
	passkey_count: number;
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

const userService = createHttpService(api);

/** Another admin's lost second factor: TOTP off and its codes deleted. */
export const resetTwoFactor = async (
	id: number,
): Promise<ServiceResponse<null>> =>
	userService.post<null>(
		`/backoffice/users/${id}/2fa/reset`,
		{},
		{ notifySuccess: true, notifyError: true },
	);

export const resetPasskeys = async (
	id: number,
): Promise<ServiceResponse<null>> =>
	userService.post<null>(
		`/backoffice/users/${id}/passkeys/reset`,
		{},
		{ notifySuccess: true, notifyError: true },
	);
