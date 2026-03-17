import { apiWithTenant } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const settingsService = createHttpService(apiWithTenant);

export interface ClinicalSettings {
	[key: string]: boolean | string;
	// Patient table columns
	show_next_appointment: boolean;
	show_diagnosis: boolean;
	show_medic: boolean;
	show_insurance: boolean;
	// Consultation form sections
	show_vital_signs: boolean;
	show_diagnoses: boolean;
	diagnosis_system: "cie11" | "cie10";
}

export interface TenantUISettings {
	clinical: ClinicalSettings;
}

export const DEFAULT_UI_SETTINGS: TenantUISettings = {
	clinical: {
		show_next_appointment: true,
		show_diagnosis: true,
		show_medic: true,
		show_insurance: true,
		show_vital_signs: true,
		show_diagnoses: true,
		diagnosis_system: "cie11",
	},
};

export const getUISettings = async (): Promise<
	ServiceResponse<TenantUISettings>
> => {
	return settingsService.get<TenantUISettings>("/tenants/settings", {
		notifyError: true,
	});
};

export const updateUISettings = async (
	payload: TenantUISettings,
): Promise<ServiceResponse<TenantUISettings>> => {
	return settingsService.put<TenantUISettings>("/tenants/settings", payload, {
		notifySuccess: true,
		notifyError: true,
	});
};
