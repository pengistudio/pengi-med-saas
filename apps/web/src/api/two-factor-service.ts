import { createHttpService, type ServiceResponse } from "@pengi/shared";
import type { LoginResponse } from "@/types/user-type";
import { apiWithTenant } from ".";

// Two-factor authentication of the signed-in user (/users/me/2fa). A wrong
// password or code answers 422, not 401, so it never signs the user out.
const twoFactorService = createHttpService(apiWithTenant);

export interface TwoFactorStatus {
	enabled: boolean;
	recovery_codes_remaining: number;
}

export interface TwoFactorSetup {
	secret: string;
	otpauth_uri: string;
}

/** Enabling ends every other session and answers a fresh one for this client. */
export interface TwoFactorEnableResponse extends LoginResponse {
	recovery_codes: string[];
}

export interface RecoveryCodesResponse {
	recovery_codes: string[];
}

export const getTwoFactorStatus = async (): Promise<
	ServiceResponse<TwoFactorStatus>
> =>
	twoFactorService.get<TwoFactorStatus>("/users/me/2fa", {
		notifyError: true,
	});

/** Starts enrollment: a new pending secret; 2FA stays off until enable. */
export const setupTwoFactor = async (
	currentPassword: string,
): Promise<ServiceResponse<TwoFactorSetup>> =>
	twoFactorService.post<TwoFactorSetup>(
		"/users/me/2fa/setup",
		{ current_password: currentPassword },
		{ notifyError: true },
	);

/**
 * Confirms the pending secret with a code. The answer carries a new session:
 * the new refresh cookie is only stored with credentials on this request
 * (apiWithTenant sends none by default), and the caller must switch to the
 * returned token, since every earlier token and refresh family is revoked.
 */
export const enableTwoFactor = async (
	code: string,
): Promise<ServiceResponse<TwoFactorEnableResponse>> =>
	twoFactorService.post<TwoFactorEnableResponse>(
		"/users/me/2fa/enable",
		{ code },
		{ withCredentials: true, notifySuccess: true, notifyError: true },
	);

/** Turns 2FA off: current password plus a TOTP or recovery code. */
export const disableTwoFactor = async (
	currentPassword: string,
	code: string,
): Promise<ServiceResponse<null>> =>
	twoFactorService.post<null>(
		"/users/me/2fa/disable",
		{ current_password: currentPassword, code },
		{ notifySuccess: true, notifyError: true },
	);

/** A new set of recovery codes; the previous set stops working. */
export const regenerateRecoveryCodes = async (
	currentPassword: string,
	code: string,
): Promise<ServiceResponse<RecoveryCodesResponse>> =>
	twoFactorService.post<RecoveryCodesResponse>(
		"/users/me/2fa/recovery-codes",
		{ current_password: currentPassword, code },
		{ notifySuccess: true, notifyError: true },
	);
