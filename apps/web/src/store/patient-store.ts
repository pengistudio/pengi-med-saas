import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { Patient } from "@/api/clinical-service";

type PatientState = {
	patient?: Patient;
	cleanPatient: () => void;
	setPatient: (patient?: Patient) => void;
	patientList: Patient[];
	setPatientList: (patientList: Patient[]) => void;
};

export const usePatientStore = create<PatientState>()(
	persist(
		(set) => ({
			patient: undefined,
			patientList: [],
			cleanPatient: () => set({ patient: undefined }),
			setPatient: (patient?: Patient) => set({ patient }),
			setPatientList: (patientList: Patient[]) => set({ patientList }),
		}),
		{ name: "patient", storage: createJSONStorage(() => sessionStorage) },
	),
);

// Selectors — pass to usePatientStore(selector) in components
export const selectPatient = (s: PatientState) => s.patient;
export const selectPatientList = (s: PatientState) => s.patientList;
export const selectSetPatient = (s: PatientState) => s.setPatient;
export const selectSetPatientList = (s: PatientState) => s.setPatientList;
export const selectCleanPatient = (s: PatientState) => s.cleanPatient;
