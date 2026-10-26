import { useMessageStore } from "@pengi/shared";
import axios, {
	type AxiosError,
	type AxiosInstance,
	type InternalAxiosRequestConfig,
} from "axios";
import { env } from "@/lib/env";
import { useSessionStore } from "@/store/session-store";

// Public API root, for the UI messages (initShared).
export const rootApi = axios.create({
	baseURL: env.VITE_API_URL,
	timeout: 10000,
	headers: { "Content-Type": "application/json" },
});

// Everything the portal calls lives under /portal. The refresh token travels
// in an httpOnly cookie, hence withCredentials.
export const portalApi = axios.create({
	baseURL: `${env.VITE_API_URL}/portal`,
	timeout: 20000,
	withCredentials: true,
	headers: { "Content-Type": "application/json" },
});

// The API answers in the interface language, not the browser's.
function sendInterfaceLanguage(client: AxiosInstance) {
	client.interceptors.request.use((config) => {
		config.headers["Accept-Language"] = useMessageStore.getState().lang || "es";
		return config;
	});
}

sendInterfaceLanguage(rootApi);
sendInterfaceLanguage(portalApi);

// Endpoints where a 401 is an answer, not an expired session.
const NO_REFRESH = ["/login", "/verify", "/refresh", "/logout"];

let refreshing: Promise<string | undefined> | undefined;

/** Trades the refresh cookie for a new access token (one call at a time). */
export function refreshAccessToken(): Promise<string | undefined> {
	refreshing ??= portalApi
		.post<{ data: { access_token: string } }>("/refresh")
		.then((res) => {
			const token = res.data.data.access_token;
			useSessionStore.getState().setAccessToken(token);
			return token;
		})
		.catch(() => {
			useSessionStore.getState().clear();
			return undefined;
		})
		.finally(() => {
			refreshing = undefined;
		});
	return refreshing;
}

portalApi.interceptors.request.use((config) => {
	const token = useSessionStore.getState().accessToken;
	if (token) config.headers.Authorization = `Bearer ${token}`;
	return config;
});

type Retriable = InternalAxiosRequestConfig & { _retried?: boolean };

// On a 401 try the refresh cookie once; if that fails the session is over
// (the store clears and the home page sends the patient back to sign in).
portalApi.interceptors.response.use(undefined, async (error: AxiosError) => {
	const config = error.config as Retriable | undefined;
	if (
		error.response?.status !== 401 ||
		!config ||
		config._retried ||
		NO_REFRESH.some((path) => config.url?.startsWith(path))
	) {
		return Promise.reject(error);
	}
	config._retried = true;
	const token = await refreshAccessToken();
	if (!token) return Promise.reject(error);
	config.headers.Authorization = `Bearer ${token}`;
	return portalApi(config);
});
