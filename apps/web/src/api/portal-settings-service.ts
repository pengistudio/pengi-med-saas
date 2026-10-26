import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from ".";

const portalSettingsService = createHttpService(apiWithTenant);

export type PortalSections = {
	appointments: boolean;
	results: boolean;
	documents: boolean;
	invoices: boolean;
};

export type PortalSettings = {
	enabled: boolean;
	sections: PortalSections;
};

export interface PortalSettingsResponse extends PortalSettings {
	portal_url: string;
}

export const getPortalSettings = async (): Promise<
	ServiceResponse<PortalSettingsResponse>
> =>
	portalSettingsService.get<PortalSettingsResponse>("/portal-settings", {
		notifyError: true,
	});

export const updatePortalSettings = async (
	settings: PortalSettings,
): Promise<ServiceResponse<PortalSettingsResponse>> =>
	portalSettingsService.put<PortalSettingsResponse>(
		"/portal-settings",
		settings,
		{ notifySuccess: true, notifyError: true },
	);
