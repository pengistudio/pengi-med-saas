import { createHttpService, type ServiceResponse } from "@pengi/shared";
import type {
	AuthenticationResponseJSON,
	PublicKeyCredentialCreationOptionsJSON,
	PublicKeyCredentialRequestOptionsJSON,
	RegistrationResponseJSON,
} from "@simplewebauthn/browser";
import { useSessionStore } from "@/store/session-store";
import type { LoginResponse } from "@/types/user-type";
import { apiWithTenant, noAuthApi } from ".";

// Passkeys (WebAuthn) of the signed-in user (/users/me/passkeys) and the
// public passkey sign-in (/auth/passkey/*). A wrong password or code answers
// 422, not 401, so it never signs the user out.
const passkeyService = createHttpService(apiWithTenant);
const loginService = createHttpService(noAuthApi);

export interface Passkey {
	id: number;
	name: string;
	created_at: string;
	last_used_at: string | null;
}

export interface PasskeyRegistrationBegin {
	challenge_id: string;
	options: PublicKeyCredentialCreationOptionsJSON;
}

export interface PasskeyLoginBegin {
	challenge_id: string;
	options: PublicKeyCredentialRequestOptionsJSON;
}

export const getPasskeys = async (): Promise<ServiceResponse<Passkey[]>> =>
	passkeyService.get<Passkey[]>("/users/me/passkeys", { notifyError: true });

/** `code` (TOTP or recovery code) is required when the user has 2FA on. */
export const beginPasskeyRegistration = async (
	currentPassword: string,
	code?: string,
): Promise<ServiceResponse<PasskeyRegistrationBegin>> =>
	passkeyService.post<PasskeyRegistrationBegin>(
		"/users/me/passkeys/register/begin",
		{ current_password: currentPassword, ...(code ? { code } : {}) },
		{ notifyError: true },
	);

export const finishPasskeyRegistration = async (
	challengeId: string,
	name: string,
	credential: RegistrationResponseJSON,
): Promise<ServiceResponse<Passkey>> => {
	const res = await passkeyService.post<Passkey>(
		"/users/me/passkeys/register/finish",
		{ challenge_id: challengeId, name, credential },
		{ notifySuccess: true, notifyError: true },
	);
	// Enrolled: a clinic requiring two-step verification lets them in now.
	if (res.success) useSessionStore.getState().setMfaRequired(false);
	return res;
};

export const deletePasskey = async (
	id: number,
): Promise<ServiceResponse<null>> =>
	passkeyService.delete<null>(`/users/me/passkeys/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});

export const beginPasskeyLogin = async (): Promise<
	ServiceResponse<PasskeyLoginBegin>
> =>
	loginService.post<PasskeyLoginBegin>(
		"/auth/passkey/begin",
		{},
		{ notifyError: true },
	);

/**
 * Answers the same session as a password login and sets the refresh cookie
 * (hence the credentials).
 */
export const loginWithPasskey = async (
	challengeId: string,
	credential: AuthenticationResponseJSON,
): Promise<ServiceResponse<LoginResponse>> =>
	loginService.post<LoginResponse>(
		"/auth/passkey/login",
		{ challenge_id: challengeId, credential },
		{ withCredentials: true, notifySuccess: true, notifyError: true },
	);
