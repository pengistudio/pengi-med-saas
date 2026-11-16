import { createHttpService, type ServiceResponse } from "@pengi/shared";
import type {
	AuthenticationResponseJSON,
	PublicKeyCredentialCreationOptionsJSON,
	PublicKeyCredentialRequestOptionsJSON,
	RegistrationResponseJSON,
} from "@simplewebauthn/browser";
import { toast } from "sonner";
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

/** `notifyError: false` for background checks that must not toast. */
export const getPasskeys = async (
	notifyError = true,
): Promise<ServiceResponse<Passkey[]>> =>
	passkeyService.get<Passkey[]>("/users/me/passkeys", { notifyError });

/** The session is not fresh: registering needs the current password again. */
export const PASSKEY_REAUTH_REQUIRED = "E-AUTH-027";

/**
 * Without `currentPassword` the API accepts a session that signed in with a
 * password or passkey less than 5 minutes ago; otherwise it answers
 * E-AUTH-027 (silently here: the caller asks for the password and retries).
 * `code` (TOTP or recovery code) is required with a password when 2FA is on.
 */
export const beginPasskeyRegistration = async (
	currentPassword?: string,
	code?: string,
): Promise<ServiceResponse<PasskeyRegistrationBegin>> => {
	const res = await passkeyService.post<PasskeyRegistrationBegin>(
		"/users/me/passkeys/register/begin",
		currentPassword
			? { current_password: currentPassword, ...(code ? { code } : {}) }
			: {},
		{ notifyError: Boolean(currentPassword) },
	);
	if (
		!res.success &&
		!currentPassword &&
		res.data?.error_code !== PASSKEY_REAUTH_REQUIRED
	) {
		toast.error(res.message);
	}
	return res;
};

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
