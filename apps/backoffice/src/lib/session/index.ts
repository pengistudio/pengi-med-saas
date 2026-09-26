import { useStore } from "zustand";
import { createHttpService } from "@/api/fetch";
import { noAuthApi } from "@/api/http-clients";
import { type AuthGateway, createSession } from "./session";

const authService = createHttpService(noAuthApi);
// The refresh token travels in an HttpOnly cookie on the API's domain.
const withCookie = { withCredentials: true };

const httpGateway: AuthGateway = {
	async login(credentials) {
		const res = await authService.post<{ token: string }>(
			"/backoffice/auth/login",
			credentials,
			{ ...withCookie, notifySuccess: true, notifyError: true },
		);
		return res.success ? res.data.token : null;
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
	return { ...state, login: session.login, logout: session.logout };
}

export { safeNext } from "./next";
export { RequireSession } from "./require-session";
export type { Credentials, Session } from "./session";
