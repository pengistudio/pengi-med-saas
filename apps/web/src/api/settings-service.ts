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
	[key: string]: unknown;
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

export const getPrescriptionTemplateStatus = async (): Promise<
	ServiceResponse<{ has_custom: boolean }>
> => {
	return settingsService.get<{ has_custom: boolean }>(
		"/clinical/prescription-template/status",
	);
};

export const uploadPrescriptionTemplate = async (
	file: File,
): Promise<ServiceResponse<{ has_custom: boolean }>> => {
	const form = new FormData();
	form.append("template", file);
	return settingsService.postForm<{ has_custom: boolean }>(
		"/clinical/prescription-template",
		form,
		{ notifySuccess: true, notifyError: true },
	);
};

export const deletePrescriptionTemplate = async (): Promise<
	ServiceResponse<{ has_custom: boolean }>
> => {
	return settingsService.delete<{ has_custom: boolean }>(
		"/clinical/prescription-template",
		{ notifySuccess: true, notifyError: true },
	);
};
