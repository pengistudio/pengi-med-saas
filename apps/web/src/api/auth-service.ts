import { createHttpService, type ServiceResponse } from "@pengi/shared";
import type {
	LoginRequest,
	LoginResponse,
	LoginResult,
} from "@/types/user-type";
import { apiWithTenant, noAuthApi } from ".";

const loginService = createHttpService(noAuthApi);

/** A session, or a 2FA challenge (see isTwoFactorChallenge). */
export const userLogin = async (
	data: LoginRequest,
): Promise<ServiceResponse<LoginResult>> => {
	return loginService.post<LoginResult>("/auth/login", data, {
		notifySuccess: true,
		notifyError: true,
	});
};

/**
 * Second step of a 2FA login: the mfa token from userLogin plus a TOTP or
 * recovery code. Answers the same session as a plain login and sets the
 * refresh cookie (hence the credentials).
 */
export const userLoginTwoFactor = async (
	mfaToken: string,
	code: string,
): Promise<ServiceResponse<LoginResponse>> => {
	return loginService.post<LoginResponse>(
		"/auth/login/2fa",
		{ mfa_token: mfaToken, code },
		{ withCredentials: true, notifySuccess: true, notifyError: true },
	);
};

export const logoutRequest = async (): Promise<ServiceResponse<null>> => {
	return loginService.post<null>(
		"/auth/logout",
		{},
		{
			withCredentials: true, // sends the refresh_token cookie so the backend can revoke it
		},
	);
};

const authedService = createHttpService(apiWithTenant);

export const changePassword = async (
	currentPassword: string,
	newPassword: string,
): Promise<ServiceResponse<null>> =>
	authedService.post<null>(
		"/users/change-password",
		{ current_password: currentPassword, new_password: newPassword },
		{ notifySuccess: true, notifyError: true },
	);

export interface RefreshTokenResponse {
	token: string;
}

export const refreshToken = async (): Promise<
	ServiceResponse<RefreshTokenResponse>
> => {
	return loginService.post<RefreshTokenResponse>(
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

export const forgotPassword = async (
	identifier: string,
): Promise<ServiceResponse<null>> => {
	return loginService.post<null>(
		"/auth/forgot-password",
		{ identifier },
		{ notifyError: true },
	);
};

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

export interface RegisterRequest extends Record<string, unknown> {
	token: string;
	company_name: string;
	username: string;
	email: string;
	password: string;
}

export interface RegisterResponse {
	user_id: number;
	username: string;
	email: string;
}

export const register = async (
	data: RegisterRequest,
): Promise<ServiceResponse<RegisterResponse>> => {
	return loginService.post<RegisterResponse>("/auth/register", data, {
		notifyError: true,
	});
};

export const verifyEmail = async (
	token: string,
): Promise<ServiceResponse<null>> => {
	return loginService.get<null>(
		`/auth/verify-email?token=${encodeURIComponent(token)}`,
		{
			notifyError: false,
		},
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

export const checkCompanySignupEmail = async (
	token: string,
	email: string,
): Promise<ServiceResponse<{ exists: boolean }>> => {
	return loginService.get<{ exists: boolean }>(
		`/auth/signup/company/check-email?token=${encodeURIComponent(token)}&email=${encodeURIComponent(email)}`,
	);
};

export interface JoinExistingCompanyRequest extends Record<string, unknown> {
	token: string;
	email: string;
	password: string;
}

/** A session, or a 2FA challenge when the account has 2FA. */
export type JoinExistingCompanyResponse = LoginResult;

export const joinCompanyWithExistingAccount = async (
	data: JoinExistingCompanyRequest,
): Promise<ServiceResponse<JoinExistingCompanyResponse>> => {
	return loginService.post<JoinExistingCompanyResponse>(
		"/auth/signup/company/join",
		data,
		{
			notifyError: true,
		},
	);
};
