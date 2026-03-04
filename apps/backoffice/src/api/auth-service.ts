import type { LoginRequest, LoginResponse } from "@/types/backoffice-user-type";
import { api, noAuthApi } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const loginService = createHttpService(noAuthApi);
const httpService = createHttpService(api);

export const userLogin = async (
	data: LoginRequest,
): Promise<ServiceResponse<LoginResponse>> => {
	return loginService.post<LoginResponse>("/backoffice/auth/login", data, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const extendSessionWithToken = async (): Promise<
	ServiceResponse<LoginResponse>
> => {
	return httpService.post<LoginResponse>("/backoffice/auth/extend");
};

export const refreshToken = async (): Promise<
	ServiceResponse<LoginResponse>
> => {
	return loginService.post<LoginResponse>(
		"/backoffice/auth/refresh",
		{},
		{
			withCredentials: true,
		},
	);
};
