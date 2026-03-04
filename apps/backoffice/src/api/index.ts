import axios from "axios";
import { useTokenStore } from "@/store/token-store";

export const api = axios.create({
	baseURL: import.meta.env.VITE_API_URL || "http://localhost:8080/api/v1",
	timeout: 10000,
	headers: {
		"Content-Type": "application/json",
	},
});

api.interceptors.request.use(
	(config) => {
		const token = useTokenStore.getState().token;

		if (token) {
			config.headers.Authorization = `Bearer ${token}`;
		}
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

		if (error.response?.status === 401 && !originalRequest._retry) {
			originalRequest._retry = true;

			try {
				const { refreshToken } = await import("@/api/auth-service");
				const result = await refreshToken();
				if (result.success) {
					useTokenStore.getState().setToken(result.data.token);
					originalRequest.headers.Authorization = `Bearer ${result.data.token}`;
					return api(originalRequest);
				}
				return Promise.reject(error);
			} catch (refreshError) {
				return Promise.reject(refreshError);
			}
		}

		return Promise.reject(error);
	},
);

export const noAuthApi = axios.create({
	baseURL: import.meta.env.VITE_API_URL || "http://localhost:8080/api/v1",
	timeout: 10000,
	headers: {
		"Content-Type": "application/json",
	},
});

noAuthApi.interceptors.request.use(
	(config) => {
		return config;
	},
	(error) => {
		return Promise.reject(error);
	},
);
