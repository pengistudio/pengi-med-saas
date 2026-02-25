import type { EnvironmentWithCompany } from "@/types/user-type";
import { api } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(api);

export const getEnvironmentsFromUser = async (
	exchangeToken: string,
): Promise<ServiceResponse<EnvironmentWithCompany[]>> => {
	return httpService.get<EnvironmentWithCompany[]>(
		`/users/environments?exchange_token=${exchangeToken}`,
	);
};
