import { createHttpService, type ServiceResponse } from "@pengi/shared";
import type { TotpSetup } from "./auth-service";
import { api } from "./index";

// The signed-in admin's own second factors (/backoffice/me/*). A wrong
// password or code answers 422, never 401, so it never ends the session.
const meService = createHttpService(api);

export interface TwoFactorStatus {
	enabled: boolean;
	recovery_codes_remaining: number;
	passkey_count: number;
	/** False when the API has no WebAuthn configuration. */
	passkeys_available: boolean;
}

export const getTwoFactorStatus = async (): Promise<
	ServiceResponse<TwoFactorStatus>
> =>
	meService.get<TwoFactorStatus>("/backoffice/me/2fa", { notifyError: true });

export const setupTwoFactor = async (
	currentPassword: string,
): Promise<ServiceResponse<TotpSetup>> =>
	meService.post<TotpSetup>(
		"/backoffice/me/2fa/setup",
		{ current_password: currentPassword },
		{ notifyError: true },
	);

export const enableTwoFactor = async (
	code: string,
): Promise<ServiceResponse<{ recovery_codes: string[] }>> =>
	meService.post<{ recovery_codes: string[] }>(
		"/backoffice/me/2fa/enable",
		{ code },
		{ notifyError: true },
	);

/** E-BO-013: the only second factor can't be removed. */
export const LAST_FACTOR = "E-BO-013";

export const disableTwoFactor = async (
	currentPassword: string,
	code: string,
): Promise<ServiceResponse<null>> =>
	meService.post<null>(
		"/backoffice/me/2fa/disable",
		{ current_password: currentPassword, code },
		{ notifySuccess: true, notifyError: true },
	);

export const regenerateRecoveryCodes = async (
	currentPassword: string,
	code: string,
): Promise<ServiceResponse<{ recovery_codes: string[] }>> =>
	meService.post<{ recovery_codes: string[] }>(
		"/backoffice/me/2fa/recovery-codes",
		{ current_password: currentPassword, code },
		{ notifyError: true },
	);
