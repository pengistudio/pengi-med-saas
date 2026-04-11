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
	// Patient form
	patient_age_input: boolean;
}

export interface KanbanSettings {
	auto_archive_delay: "never" | "1_day" | "1_week" | "2_weeks" | "1_month";
}

export interface TenantUISettings {
	[key: string]: unknown;
	clinical: ClinicalSettings;
	kanban?: KanbanSettings;
}

export interface EnabledFeatures extends Record<string, unknown> {
	clinical?: boolean;
	billing?: boolean;
	team?: boolean;
	kanban?: boolean;
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
		patient_age_input: false,
	},
	kanban: {
		auto_archive_delay: "never",
	},
};

export const DEFAULT_ENABLED_FEATURES: EnabledFeatures = {
	clinical: true,
	billing: true,
	team: true,
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

export const getEnabledFeatures = async (): Promise<
	ServiceResponse<EnabledFeatures>
> => {
	return settingsService.get<EnabledFeatures>("/tenants/features", {
		notifyError: true,
	});
};

export const updateEnabledFeatures = async (
	payload: EnabledFeatures,
): Promise<ServiceResponse<EnabledFeatures>> => {
	return settingsService.put<EnabledFeatures>("/tenants/features", payload, {
		notifySuccess: true,
		notifyError: true,
	});
};
