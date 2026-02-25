import type { LoginRequest, LoginResponse } from "@/types/user-type";
import { api, noAuthApi } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const loginService = createHttpService(noAuthApi);
const httpService = createHttpService(api);

export const userLogin = async (
	data: LoginRequest,
): Promise<ServiceResponse<LoginResponse>> => {
	return loginService.post<LoginResponse>("/auth/login", data, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const extendSessionWithToken = async (): Promise<
	ServiceResponse<LoginResponse>
> => {
	// httpService automatically attaches the token via intercepts
	// We no longer need to pass the header manually because it will be intercepted
	return httpService.post<LoginResponse>("/auth/extend");
};

export const refreshToken = async (): Promise<
	ServiceResponse<LoginResponse>
> => {
	return loginService.post<LoginResponse>(
		"/auth/refresh",
		{},
		{
			withCredentials: true, // necessary if refresh_token is in HttpOnly cookie
		},
	);
};
