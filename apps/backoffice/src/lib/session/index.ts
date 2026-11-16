import { createHttpService } from "@pengi/shared";
import { useStore } from "zustand";
import { noAuthApi } from "@/api/http-clients";
import { type AuthGateway, createSession, type LoginOutcome } from "./session";
import { userIdOf } from "./user-id";

interface LoginResponse {
	token?: string;
	two_factor_required?: boolean;
	mfa_token?: string;
	totp?: boolean;
	passkey?: boolean;
	enrollment_required?: boolean;
	enrollment_token?: string;
	passkeys_available?: boolean;
}

function toOutcome(data: LoginResponse): LoginOutcome {
	if (data.two_factor_required && data.mfa_token) {
		return {
			kind: "two_factor",
			mfaToken: data.mfa_token,
			totp: data.totp ?? false,
			passkey: data.passkey ?? false,
		};
	}
	if (data.enrollment_required && data.enrollment_token) {
		return {
			kind: "enrollment",
			enrollmentToken: data.enrollment_token,
			passkeysAvailable: data.passkeys_available ?? false,
		};
	}
	return data.token
		? { kind: "session", token: data.token }
		: { kind: "failed" };
}

const authService = createHttpService(noAuthApi);
// The refresh token travels in an HttpOnly cookie on the API's domain.
const withCookie = { withCredentials: true };

const httpGateway: AuthGateway = {
	async login(credentials) {
		const res = await authService.post<LoginResponse>(
			"/backoffice/auth/login",
			credentials,
			{ ...withCookie, notifyError: true },
		);
		if (!res.success) return { kind: "failed" };
		return toOutcome(res.data);
	},
	async refresh() {
		const res = await authService.post<{ token: string }>(
			"/backoffice/auth/refresh",
			{},
			withCookie,
		);
		return res.success ? res.data.token : null;
	},
	async logout() {
		await authService.post<null>("/backoffice/auth/logout", {}, withCookie);
	},
};

/** The backoffice's session. `api` requests are authorized through it. */
export const session = createSession(httpGateway);

export function useSession() {
	const state = useStore(session.store);
	return {
		...state,
		userId: userIdOf(state.token),
		login: session.login,
		establish: session.establish,
		logout: session.logout,
	};
}

export {
	initialStep,
	type LoginEvent,
	type LoginStep,
	nextStep,
} from "./login-flow";
export { safeNext } from "./next";
export { RequireSession } from "./require-session";
export type { Credentials, LoginOutcome, Session } from "./session";
export { userIdOf } from "./user-id";
