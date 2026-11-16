import { createHttpService, type ServiceResponse } from "@pengi/shared";
import type { RegistrationResponseJSON } from "@simplewebauthn/browser";
import type {
	BackofficePasskey,
	PasskeyRegistrationBegin,
} from "./auth-service";
import { api } from "./index";

// Passkeys (WebAuthn) of the signed-in admin (/backoffice/me/passkeys).
const passkeyService = createHttpService(api);

export type { BackofficePasskey as Passkey };

/** The session is not fresh (a passkey-only admin): sign in again first. */
export const PASSKEY_REAUTH_REQUIRED = "E-AUTH-027";

export const getPasskeys = async (): Promise<
	ServiceResponse<BackofficePasskey[]>
> =>
	passkeyService.get<BackofficePasskey[]>("/backoffice/me/passkeys", {
		notifyError: true,
	});

/** The password is always required; `code` (TOTP or recovery) when 2FA is on. */
export const beginPasskeyRegistration = async (
	currentPassword: string,
	code?: string,
): Promise<ServiceResponse<PasskeyRegistrationBegin>> =>
	passkeyService.post<PasskeyRegistrationBegin>(
		"/backoffice/me/passkeys/register/begin",
		{ current_password: currentPassword, ...(code ? { code } : {}) },
		{ notifyError: true },
	);

export const finishPasskeyRegistration = async (
	challengeId: string,
	name: string,
	credential: RegistrationResponseJSON,
): Promise<ServiceResponse<BackofficePasskey>> =>
	passkeyService.post<BackofficePasskey>(
		"/backoffice/me/passkeys/register/finish",
		{ challenge_id: challengeId, name, credential },
		{ notifySuccess: true, notifyError: true },
	);

export const deletePasskey = async (
	id: number,
): Promise<ServiceResponse<null>> =>
	passkeyService.delete<null>(`/backoffice/me/passkeys/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});
