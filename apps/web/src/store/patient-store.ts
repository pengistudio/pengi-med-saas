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

const persistPatient = persist<PatientState>(
	(set) => ({
		patient: undefined,
		patientList: [],
		cleanPatient: () => set({ patient: undefined }),
		setPatient: (patient?: Patient) => set({ patient }),
		setPatientList: (patientList: Patient[]) => set({ patientList }),
	}),
	{ name: "patient", storage: createJSONStorage(() => sessionStorage) },
);

export const usePatientStore = create(persistPatient);
