// TypeScript interfaces for UI settings (mirror backend UISettings)

export type AutoArchiveDelay =
	| "never"
	| "1_day"
	| "1_week"
	| "2_weeks"
	| "1_month";

export type DiagnosisSystem = "cie11" | "cie10";

export interface KanbanSettings {
	auto_archive_delay: AutoArchiveDelay;
}

export interface ClinicalSettings {
	show_next_appointment: boolean;
	show_diagnosis: boolean;
	show_medic: boolean;
	show_insurance: boolean;
	show_vital_signs: boolean;
	show_diagnoses: boolean;
	diagnosis_system: DiagnosisSystem;
	patient_age_input: boolean;
}

export interface TenantUISettings {
	clinical: ClinicalSettings;
	kanban: KanbanSettings;
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
