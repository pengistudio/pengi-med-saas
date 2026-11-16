import { createHttpService, type ServiceResponse } from "@pengi/shared";
import type {
	AuthenticationResponseJSON,
	PublicKeyCredentialCreationOptionsJSON,
	PublicKeyCredentialRequestOptionsJSON,
	RegistrationResponseJSON,
} from "@simplewebauthn/browser";
import { noAuthApi } from "./http-clients";

// The login phase: the mfa and enrollment tokens travel in the body, and
// the client has no 401-refresh interceptor, so a wrong or expired one never
// looks like an expired session. Answers that start a session also set the
// refresh cookie (hence withCredentials).
const authService = createHttpService(noAuthApi);
const withCookie = { withCredentials: true };
const notifyError = { notifyError: true };

export interface SessionData {
	token: string;
	user_id: number;
}

export interface TotpSetup {
	secret: string;
	otpauth_uri: string;
}

export interface BackofficePasskey {
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

/** The mfa or enrollment token expired or no longer matches the account. */
export const INVALID_MFA_TOKEN = "E-AUTH-022";

/** Second step: a 6-digit code or a recovery code (XXXX-XXXX-XXXX-XXXX). */
export const loginTwoFactor = async (
	mfaToken: string,
	code: string,
): Promise<ServiceResponse<SessionData>> =>
	authService.post<SessionData>(
		"/backoffice/auth/login/2fa",
		{ mfa_token: mfaToken, code },
		{ ...withCookie, ...notifyError },
	);

export const beginTotpEnrollment = async (
	enrollmentToken: string,
): Promise<ServiceResponse<TotpSetup>> =>
	authService.post<TotpSetup>(
		"/backoffice/auth/enroll/totp/begin",
		{ enrollment_token: enrollmentToken },
		notifyError,
	);

export const finishTotpEnrollment = async (
	enrollmentToken: string,
	code: string,
): Promise<ServiceResponse<SessionData & { recovery_codes: string[] }>> =>
	authService.post<SessionData & { recovery_codes: string[] }>(
		"/backoffice/auth/enroll/totp/finish",
		{ enrollment_token: enrollmentToken, code },
		{ ...withCookie, ...notifyError },
	);

export const beginPasskeyEnrollment = async (
	enrollmentToken: string,
): Promise<ServiceResponse<PasskeyRegistrationBegin>> =>
	authService.post<PasskeyRegistrationBegin>(
		"/backoffice/auth/enroll/passkey/begin",
		{ enrollment_token: enrollmentToken },
		notifyError,
	);

export const finishPasskeyEnrollment = async (
	enrollmentToken: string,
	challengeId: string,
	name: string,
	credential: RegistrationResponseJSON,
): Promise<ServiceResponse<SessionData & { passkey: BackofficePasskey }>> =>
	authService.post<SessionData & { passkey: BackofficePasskey }>(
		"/backoffice/auth/enroll/passkey/finish",
		{
			enrollment_token: enrollmentToken,
			challenge_id: challengeId,
			name,
			credential,
		},
		{ ...withCookie, ...notifyError },
	);

export const beginPasskeyLogin = async (): Promise<
	ServiceResponse<PasskeyLoginBegin>
> =>
	authService.post<PasskeyLoginBegin>(
		"/backoffice/auth/passkey/begin",
		{},
		notifyError,
	);

export const loginWithPasskey = async (
	challengeId: string,
	credential: AuthenticationResponseJSON,
): Promise<ServiceResponse<SessionData>> =>
	authService.post<SessionData>(
		"/backoffice/auth/passkey/login",
		{ challenge_id: challengeId, credential },
		{ ...withCookie, ...notifyError },
	);
