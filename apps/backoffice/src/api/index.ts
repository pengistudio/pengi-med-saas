import { refreshToken } from "@/api/auth-service";
import { useTokenStore } from "@/store/token-store";
import { api, noAuthApi } from "./http-clients";

export { api, noAuthApi };

api.interceptors.request.use(
	(config) => {
		const token = useTokenStore.getState().token;
		if (token) {
			config.headers.Authorization = `Bearer ${token}`;
		}
		return config;
	},
	(error) => Promise.reject(error),
);

api.interceptors.response.use(
	(response) => response,
	async (error) => {
		const originalRequest = error.config;

		if (error.response?.status === 401 && !originalRequest._retry) {
			originalRequest._retry = true;

			try {
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

noAuthApi.interceptors.request.use(
	(config) => config,
	(error) => Promise.reject(error),
);
