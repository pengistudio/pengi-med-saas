import { useMessageStore } from "@pengi/shared";
import axios, { type AxiosInstance } from "axios";

const BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8000/api/v1";

export const api = axios.create({
	baseURL: BASE_URL,
	timeout: 10000,
	headers: { "Content-Type": "application/json" },
});

export const noAuthApi = axios.create({
	baseURL: BASE_URL,
	timeout: 10000,
	headers: { "Content-Type": "application/json" },
});

// The API answers in the language of Accept-Language; send the interface
// language, not the browser's, so toasts match the rest of the UI.
function sendInterfaceLanguage(client: AxiosInstance) {
	client.interceptors.request.use((config) => {
		config.headers["Accept-Language"] = useMessageStore.getState().lang || "es";
		return config;
	});
}

sendInterfaceLanguage(api);
sendInterfaceLanguage(noAuthApi);
