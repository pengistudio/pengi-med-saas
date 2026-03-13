import axios, {
	type AxiosError,
	type AxiosInstance,
	type AxiosRequestConfig,
	type AxiosResponse,
} from "axios";
import { toast } from "sonner";

// From backend core_errors/codes.go and envelope/response.go
export type ResponseError = {
	error_code: string;
	error_message: string;
};

// Base response properties
type BaseResponse = {
	code: number; // Matches Go 'int'
	message: string;
};

export type SuccessResponse<T> = BaseResponse & {
	success: true;
	data: T;
	filename?: string;
};

export type ErrorResponse = BaseResponse & {
	success: false;
	data: ResponseError;
};

export type ServiceResponse<T> = SuccessResponse<T> | ErrorResponse;

type BackendResponse<T> = {
	code: number;
	message: string;
	data: T;
};

export interface BaseModel {
	ID: number;
	CreatedAt: string; // formato ISO string
	UpdatedAt: string;
	DeletedAt?: string | null;
}

export interface CustomAxiosRequestConfig extends AxiosRequestConfig {
	/**
	 * If true, shows a toast notification with the backend message on success.
	 * If string, shows that specific message on success.
	 */
	notifySuccess?: boolean | string;
	/**
	 * If true, shows a toast notification with the backend message on error.
	 * If string, shows that specific message on error.
	 */
	notifyError?: boolean | string;
}

export class HttpService {
	private client: AxiosInstance;

	constructor(client: AxiosInstance) {
		this.client = client;
	}

	async get<T>(
		url: string,
		config?: CustomAxiosRequestConfig,
	): Promise<ServiceResponse<T>> {
		return this.request<T>(() => this.client.get(url, config), config);
	}

	async post<T>(
		url: string,
		data?: Record<string, unknown> | FormData,
		config?: CustomAxiosRequestConfig,
	): Promise<ServiceResponse<T>> {
		return this.request<T>(() => this.client.post(url, data, config), config);
	}

	async put<T>(
		url: string,
		data?: Record<string, unknown> | FormData,
		config?: CustomAxiosRequestConfig,
	): Promise<ServiceResponse<T>> {
		return this.request<T>(() => this.client.put(url, data, config), config);
	}

	async postForm<T>(
		url: string,
		data: FormData,
		config?: CustomAxiosRequestConfig,
	): Promise<ServiceResponse<T>> {
		return this.request<T>(
			() =>
				this.client.post(url, data, {
					...config,
					headers: {
						...config?.headers,
						"Content-Type": "multipart/form-data",
					},
				}),
			config,
		);
	}

	async putForm<T>(
		url: string,
		data: FormData,
		config?: CustomAxiosRequestConfig,
	): Promise<ServiceResponse<T>> {
		return this.request<T>(
			() =>
				this.client.put(url, data, {
					...config,
					headers: {
						...config?.headers,
						"Content-Type": "multipart/form-data",
					},
				}),
			config,
		);
	}

	async delete<T>(
		url: string,
		config?: CustomAxiosRequestConfig,
	): Promise<ServiceResponse<T>> {
		return this.request<T>(() => this.client.delete(url, config), config);
	}

	private async request<T>(
		fn: () => Promise<AxiosResponse<unknown>>,
		config?: CustomAxiosRequestConfig,
	): Promise<ServiceResponse<T>> {
		try {
			const response = await fn();

			// Handle blob responses specifically, as they don't have the standard BaseResponse structure
			if (response.data instanceof Blob) {
				const message = "Success";
				if (config?.notifySuccess) {
					toast.success(
						typeof config.notifySuccess === "string"
							? config.notifySuccess
							: message,
					);
				}
				const disposition: string =
					response.headers["content-disposition"] ?? "";
				const filenameMatch = disposition.match(/filename=([^;]+)/);
				const filename = filenameMatch ? filenameMatch[1].trim() : undefined;
				return {
					success: true,
					code: response.status,
					message: message,
					data: response.data as unknown as T,
					filename,
				};
			}

			const responseData = response.data as BackendResponse<T>;

			const message = responseData.message || "Success";

			if (config?.notifySuccess) {
				toast.success(
					typeof config.notifySuccess === "string"
						? config.notifySuccess
						: message,
				);
			}

			// Backend returns { code: int, message: string, data: T }
			// If axios throws no error, it's a 2xx response
			return {
				success: true,
				code: responseData.code || response.status,
				message: message,
				data: responseData.data,
			};
		} catch (error: unknown) {
			let errorObj: ErrorResponse;

			if (axios.isAxiosError(error)) {
				const axiosError = error as AxiosError<BackendResponse<ResponseError>>;
				const errorResponse = axiosError.response;
				const responseData = errorResponse?.data;
				const errorMessage =
					responseData?.message || axiosError.message || "Unknown error";

				errorObj = {
					success: false,
					code: responseData?.code || errorResponse?.status || 500,
					message: errorMessage,
					data: responseData?.data || {
						error_code: "UNKNOWN",
						error_message: axiosError.message,
					},
				};
			} else {
				errorObj = {
					success: false,
					code: 500,
					message: "Unknown error",
					data: {
						error_code: "UNKNOWN",
						error_message: "Unknown error",
					},
				};
			}

			if (config?.notifyError) {
				toast.error(
					typeof config.notifyError === "string"
						? config.notifyError
						: errorObj.data.error_message,
				);
			}

			return errorObj;
		}
	}
}

// Export a default instance if needed, or allow creating one with specific axios instance
export const createHttpService = (client: AxiosInstance) =>
	new HttpService(client);
