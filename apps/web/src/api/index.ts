import axios from "axios";
import { useMessageStore } from "@/store/message-store";
import { useSessionStore } from "@/store/session-store";
import { useTokenStore } from "@/store/token-store";

export const api = axios.create({
	baseURL: import.meta.env.VITE_API_URL || "http://localhost:8080/api/v1",
	headers: {
		"Content-Type": "application/json",
	},
});

export const noAuthApi = axios.create({
	baseURL: import.meta.env.VITE_API_URL || "http://localhost:8080/api/v1",
	headers: {
		"Content-Type": "application/json",
	},
	withCredentials: true,
});

export const apiWithTenant = axios.create({
	baseURL: import.meta.env.VITE_API_URL || "http://localhost:8080/api/v1",
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

api.interceptors.response.use(
	(response) => response,
	async (error) => {
		const originalRequest = error.config;

		// Si el error es 401 y no hemos intentado refrescar el token aún
		if (error.response?.status === 401 && !originalRequest._retry) {
			originalRequest._retry = true;
			// Aquí podrías implementar la lógica de refresh token si la tienes
			// Por ahora solo rechazamos
		}
		return Promise.reject(error);
	},
);
