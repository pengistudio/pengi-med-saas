import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from ".";

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

/** Automatic no-show marking limits (minutes after the end time). */
export const NO_SHOW_MINUTES = { min: 5, max: 240, default: 30 } as const;

export interface AgendaSettings {
	/** Mark as no-show the appointments still pending after their end time. */
	auto_no_show: boolean;
	/** Minutes after the end time (NO_SHOW_MINUTES). */
	no_show_after_minutes: number;
	/** When it was turned on; set by the server, ignored when sent. */
	auto_no_show_since?: string | null;
}

/** Off: existing clinics never get no-shows marked until they turn it on. */
export const DEFAULT_AGENDA_SETTINGS: AgendaSettings = {
	auto_no_show: false,
	no_show_after_minutes: NO_SHOW_MINUTES.default,
	auto_no_show_since: null,
};

export interface TenantUISettings {
	[key: string]: unknown;
	clinical: ClinicalSettings;
	kanban?: KanbanSettings;
	/** Absent in answers from an API without it (and in old local copies). */
	agenda?: AgendaSettings;
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
	agenda: DEFAULT_AGENDA_SETTINGS,
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
