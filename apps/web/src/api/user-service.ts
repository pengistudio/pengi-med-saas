import type { EnvironmentWithCompany } from "@/types/user-type";
import { api } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(api);

export const getEnvironmentsFromUser = async (
	exchangeToken: string,
): Promise<ServiceResponse<EnvironmentWithCompany[]>> => {
	return httpService.get<EnvironmentWithCompany[]>(
		`/users/environments?exchange_token=${exchangeToken}`,
	);
};

// ─── Profile API ─────────────────────────────────────────────────────────────

export interface ProfileData {
	user_id: number;
	user_name: string;
	email: string;
	environment_id: number;
	environment_name: string;
	role: string;
	legal_name: string;
	trade_name: string;
}

export type UpdateProfilePayload = {
	email?: string;
	environment_name?: string;
};

export const getProfile = async (
	environmentId: number,
): Promise<ServiceResponse<ProfileData>> => {
	return httpService.get<ProfileData>(
		`/users/profile?environment_id=${environmentId}`,
		{ notifyError: true },
	);
};

export const updateProfile = async (
	environmentId: number,
	payload: UpdateProfilePayload,
): Promise<ServiceResponse<ProfileData>> => {
	return httpService.put<ProfileData>(
		`/users/profile?environment_id=${environmentId}`,
		payload,
		{ notifySuccess: true, notifyError: true },
	);
};
