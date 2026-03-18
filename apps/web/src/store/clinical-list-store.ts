import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

type PatientSortValue =
	| "created_at_desc"
	| "created_at_asc"
	| "last_name_asc"
	| "last_name_desc";

interface ClinicalListState {
	patientSortValue: PatientSortValue;
	setPatientSortValue: (value: PatientSortValue) => void;
}

export const useClinicalListStore = create<ClinicalListState>()(
	persist(
		(set) => ({
			patientSortValue: "created_at_desc",
			setPatientSortValue: (value) => set({ patientSortValue: value }),
		}),
		{
			name: "clinical-list-storage",
			storage: createJSONStorage(() => localStorage),
		},
	),
);
