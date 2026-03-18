import type { LoginRequest, LoginResponse } from "@/types/user-type";
import { api, noAuthApi } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const loginService = createHttpService(noAuthApi);
const httpService = createHttpService(api);

export const userLogin = async (
	data: LoginRequest,
): Promise<ServiceResponse<LoginResponse>> => {
	return loginService.post<LoginResponse>("/auth/login", data, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const extendSessionWithToken = async (): Promise<
	ServiceResponse<LoginResponse>
> => {
	// httpService automatically attaches the token via intercepts
	// We no longer need to pass the header manually because it will be intercepted
	return httpService.post<LoginResponse>("/auth/extend");
};

export const refreshToken = async (): Promise<
	ServiceResponse<LoginResponse>
> => {
	return loginService.post<LoginResponse>(
		"/auth/refresh",
		{},
		{
			withCredentials: true, // necessary if refresh_token is in HttpOnly cookie
		},
	);
};

export interface CompanySignupRequest extends Record<string, unknown> {
	token: string;
	name: string;
	user_name: string;
	email: string;
	password: string;
}

export interface CompanySignupResponse {
	user_id: number;
	username: string;
	email: string;
}

export const resetPassword = async (
	token: string,
	newPassword: string,
): Promise<ServiceResponse<null>> => {
	return loginService.post<null>(
		"/auth/reset-password",
		{ token, new_password: newPassword },
		{ notifySuccess: true, notifyError: true },
	);
};

export const companySignup = async (
	data: CompanySignupRequest,
): Promise<ServiceResponse<CompanySignupResponse>> => {
	return loginService.post<CompanySignupResponse>(
		"/auth/signup/company",
		data,
		{
			notifySuccess: true,
			notifyError: true,
		},
	);
};
