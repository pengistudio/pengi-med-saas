import { apiWithTenant } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const integrationService = createHttpService(apiWithTenant);

export interface GoogleIntegrationStatus {
	connected: boolean;
	google_calendar_id?: string;
}

export const getGoogleAuthUrl = async (): Promise<
	ServiceResponse<{ url: string }>
> => {
	return integrationService.get<{ url: string }>(
		"/integrations/google/auth-url",
		{ notifyError: true },
	);
};

export const getGoogleIntegrationStatus = async (): Promise<
	ServiceResponse<GoogleIntegrationStatus>
> => {
	return integrationService.get<GoogleIntegrationStatus>(
		"/integrations/google/status",
	);
};

export const disconnectGoogle = async (): Promise<ServiceResponse<null>> => {
	return integrationService.delete<null>("/integrations/google/disconnect", {
		notifySuccess: true,
		notifyError: true,
	});
};
