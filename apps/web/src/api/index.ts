import axios, { type InternalAxiosRequestConfig } from "axios";
import { toast } from "sonner";
import { useMessageStore } from "@/store/message-store";
import { useSessionStore } from "@/store/session-store";
import { useTokenStore } from "@/store/token-store";
import { useUserStore } from "@/store/user-store";

type RetryableConfig = InternalAxiosRequestConfig & { _retry?: boolean };

// Called when a 401 is received at runtime (token revoked or expired server-side).
// Uses store getState() directly since interceptors run outside React.
let sessionExpiredHandled = false;
export function resetSessionExpiredFlag() {
	sessionExpiredHandled = false;
}
function handleSessionExpired() {
	if (sessionExpiredHandled || !useTokenStore.getState().token) return;
	sessionExpiredHandled = true;

	const messages = useMessageStore.getState().messages;
	toast.error(messages["session.expired"] ?? "Sesión expirada");

	localStorage.clear();
	sessionStorage.clear();
	useTokenStore.getState().setToken(undefined);
	useUserStore.getState().clean();
	window.location.href = "/login";
}

export const api = axios.create({
	baseURL: import.meta.env.VITE_API_URL || "http://localhost:8000/api/v1",
	headers: {
		"Content-Type": "application/json",
	},
});

export const noAuthApi = axios.create({
	baseURL: import.meta.env.VITE_API_URL || "http://localhost:8000/api/v1",
	headers: {
		"Content-Type": "application/json",
	},
	withCredentials: true,
});

export const apiWithTenant = axios.create({
	baseURL: import.meta.env.VITE_API_URL || "http://localhost:8000/api/v1",
	headers: {
		"Content-Type": "application/json",
	},
});

apiWithTenant.interceptors.request.use(
	(config) => {
		const token = useTokenStore.getState().token;
		const lang = useMessageStore.getState().lang || "es";
		const tenant = useSessionStore.getState().environment?.tenant_slug;

		if (token) {
			config.headers.Authorization = `Bearer ${token}`;
		}
		config.headers["Accept-Language"] = lang;
		config.headers["X-Tenant-Slug"] = tenant;
		return config;
	},
	(error) => {
		return Promise.reject(error);
	},
);

api.interceptors.request.use(
	(config) => {
		const token = useTokenStore.getState().token;
		const lang = useMessageStore.getState().lang || "es";

		if (token) {
			config.headers.Authorization = `Bearer ${token}`;
		}
		config.headers["Accept-Language"] = lang;
		return config;
	},
	(error) => {
		return Promise.reject(error);
	},
);

noAuthApi.interceptors.request.use(
	(config) => {
		const lang = useMessageStore.getState().lang || "es";
		config.headers["Accept-Language"] = lang;
		return config;
	},
	(error) => {
		return Promise.reject(error);
	},
);

// Dedupe concurrent 401s onto a single /auth/refresh call so a burst of
// requests hitting an expired access token don't each trigger their own
// refresh race. Uses noAuthApi directly (not the auth-service wrapper) to
// avoid a circular import between this module and auth-service.ts, which
// itself imports api/noAuthApi from here.
let refreshPromise: Promise<string | null> | null = null;

function silentRefresh(): Promise<string | null> {
	if (!refreshPromise) {
		refreshPromise = noAuthApi
			.post("/auth/refresh", {}, { withCredentials: true })
			.then((res) => {
				const newToken = res.data?.data?.token as string | undefined;
				if (!newToken) return null;
				useTokenStore.getState().setToken(newToken);
				return newToken;
			})
			.catch(() => null)
			.finally(() => {
				refreshPromise = null;
			});
	}
	return refreshPromise;
}

function make401Interceptor(instance: ReturnType<typeof axios.create>) {
	instance.interceptors.response.use(
		(response) => response,
		(error) => {
			const status = error.response?.status;
			const config = error.config as RetryableConfig | undefined;

			if (status === 401) {
				if (config && !config._retry) {
					config._retry = true;
					return silentRefresh().then((newToken) => {
						if (newToken) {
							config.headers = config.headers ?? {};
							config.headers.Authorization = `Bearer ${newToken}`;
							return instance(config);
						}
						handleSessionExpired();
						return Promise.reject(error);
					});
				}
				handleSessionExpired();
			}

			if (
				status === 403 &&
				error.response?.data?.data?.error_code === "E-BO-007"
			) {
				useSessionStore.getState().setSubscriptionExpired(true);
			}
			return Promise.reject(error);
		},
	);
}

make401Interceptor(api);
make401Interceptor(apiWithTenant);
