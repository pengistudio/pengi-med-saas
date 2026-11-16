import type {
	AxiosError,
	AxiosInstance,
	InternalAxiosRequestConfig,
} from "axios";
import { createStore } from "zustand/vanilla";

export type Credentials = { user_name: string; password: string };

/**
 * What the password step answers: never a session by itself for an admin with
 * the second factor policy on (the API asks for a code or an enrollment), but
 * the gateway may still hand one over.
 */
export type LoginOutcome =
	| { kind: "failed" }
	| { kind: "session"; token: string }
	| { kind: "two_factor"; mfaToken: string; totp: boolean; passkey: boolean }
	| { kind: "enrollment"; enrollmentToken: string; passkeysAvailable: boolean };

/** The backoffice auth API. Tokens are returned, null means "no". */
export interface AuthGateway {
	login(credentials: Credentials): Promise<LoginOutcome>;
	/** A new access token from the refresh cookie, or null if there is none. */
	refresh(): Promise<string | null>;
	logout(): Promise<void>;
}

export type SessionState = {
	status: "restoring" | "authenticated" | "anonymous";
	/** Anonymous because the session could not be renewed (not a logout). */
	expired: boolean;
	token?: string;
};

type RetriableConfig = InternalAxiosRequestConfig & { _sessionRetry?: boolean };

/**
 * The backoffice session: the access token lives only in memory; on page load
 * it is restored from the refresh cookie, and a 401 triggers one shared
 * refresh after which the failed requests are retried. If the refresh fails
 * the session expires.
 */
export function createSession(gateway: AuthGateway) {
	const store = createStore<SessionState>(() => ({
		status: "restoring",
		expired: false,
	}));

	const authenticate = (token: string) =>
		store.setState({ status: "authenticated", expired: false, token });
	const clear = (expired: boolean) =>
		store.setState({ status: "anonymous", expired, token: undefined });

	let refreshing: Promise<string | null> | null = null;
	const refreshOnce = () => {
		refreshing ??= gateway.refresh().finally(() => {
			refreshing = null;
		});
		return refreshing;
	};

	return {
		store,

		async restore() {
			const token = await refreshOnce();
			if (token) authenticate(token);
			else clear(false);
		},

		/** The password step; it signs in only when the API answers a session. */
		async login(credentials: Credentials): Promise<LoginOutcome> {
			const outcome = await gateway.login(credentials);
			if (outcome.kind === "session") authenticate(outcome.token);
			return outcome;
		},

		/** Signs in with the token a finished second step answered. */
		establish(token: string) {
			authenticate(token);
		},

		async logout() {
			try {
				await gateway.logout();
			} finally {
				clear(false);
			}
		},

		/** Authorizes the client's requests and renews the session on 401. */
		attach(client: AxiosInstance) {
			client.interceptors.request.use((config) => {
				const { token } = store.getState();
				if (token) config.headers.Authorization = `Bearer ${token}`;
				return config;
			});

			client.interceptors.response.use(undefined, async (error: AxiosError) => {
				const config = error.config as RetriableConfig | undefined;
				if (error.response?.status !== 401 || !config || config._sessionRetry) {
					throw error;
				}
				config._sessionRetry = true;

				const sentWith = config.headers.Authorization;
				const current = store.getState().token;
				// Another request already renewed the session: just retry.
				let token =
					current && sentWith !== `Bearer ${current}` ? current : undefined;
				if (!token) {
					token = (await refreshOnce()) ?? undefined;
					if (!token) {
						if (store.getState().status !== "anonymous") clear(true);
						throw error;
					}
					authenticate(token);
				}
				config.headers.Authorization = `Bearer ${token}`;
				return client(config);
			});
		},
	};
}

export type Session = ReturnType<typeof createSession>;
