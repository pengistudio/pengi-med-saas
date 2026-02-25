import { apiWithTenant } from ".";
import {
	type BaseModel,
	createHttpService,
	type ServiceResponse,
} from "./fetch";

const clinicalService = createHttpService(apiWithTenant);

export interface Patient extends BaseModel {
	tenant_id: number;
	document: string;
	phone: string;
	first_name: string;
	last_name: string;
	full_name?: string;
	birth_date: string;
	institution: string;
	gender: string;
	notes: string;
	insurance: string;
	medic: string;
	diagnosis: string;
	critical: boolean;
	app: string;
	apf: string;
	apqx: string;
	medical_records?: Array<{
		next_appointment_date?: string | null;
	}>;
}

export const getAllPatientsWithLastFollowUp = async (): Promise<
	ServiceResponse<Patient[]>
> => {
	// The token is handled by interceptors in apiWithTenant
	return clinicalService.get<Patient[]>("/clinical/patients");
};

export const deleteMultiplePatients = async (
	ids: number[],
): Promise<ServiceResponse<Patient[]>> => {
	// Our API has a DeletePatient endpoint (DELETE /clinical/patients/:id)
	// We map the calls and then fetch patients again to simulate "delete multiple and return list"
	const promises = ids.map((id) =>
		clinicalService.delete<Patient>(`/clinical/patients/${id}`, {
			notifySuccess: true,
			notifyError: true,
		}),
	);
	const results = await Promise.all(promises);

	const failed = results.find((r) => !r.success);
	if (failed) return failed as ServiceResponse<Patient[]>;

	// if all succeed, fetch freshest map
	return getAllPatientsWithLastFollowUp();
};

export const downloadPatientReport = async (id: number): Promise<void> => {
	const response = await clinicalService.get<Blob>(
		`/clinical/patients/${id}/report`,
		{
			responseType: "blob",
			notifyError: true,
		},
	);

	if (response.success && response.data) {
		const url = window.URL.createObjectURL(new Blob([response.data]));
		const link = document.createElement("a");
		link.href = url;
		link.setAttribute("download", `patient_report_${id}.pdf`);
		link.click();
		link.remove();
	}
};

export type CreatePatientPayload = {
	document: string;
	phone?: string;
	first_name: string;
	last_name: string;
	birth_date?: Date;
	next_appointment_date?: Date;
	next_appointment_start_time?: string;
	next_appointment_end_time?: string;
	institution: string;
	gender?: string;
	notes?: string;
	insurance?: string;
	medic: string;
	diagnosis?: string;
	app?: string;
	apf?: string;
	apqx?: string;
};

export const createPatient = async (
	payload: CreatePatientPayload,
): Promise<ServiceResponse<Patient>> => {
	return clinicalService.post<Patient>("/clinical/patients", payload, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const getPatientById = async (
	id: number,
): Promise<ServiceResponse<Patient>> => {
	return clinicalService.get<Patient>(`/clinical/patients/${id}`, {
		notifyError: true,
	});
};

export const updatePatient = async (
	id: number,
	payload: Partial<CreatePatientPayload>,
): Promise<ServiceResponse<Patient>> => {
	return clinicalService.put<Patient>(`/clinical/patients/${id}`, payload, {
		notifySuccess: true,
		notifyError: true,
	});
};

export interface Institution extends BaseModel {
	name: string;
}

export const getAllInstitutions = async (): Promise<
	ServiceResponse<Institution[]>
> => {
	return clinicalService.get<Institution[]>("/clinical/institutions", {
		notifyError: true,
	});
};

export interface MedicalRecord extends BaseModel {
	patient_id: number;
	date: string;
	next_appointment_date?: string | null;
	next_appointment_start_time?: string | null;
	next_appointment_end_time?: string | null;
	motive: string;
	observation: string;
	soap_record?: {
		subjective: string;
		objective: string;
		assessment: string;
		plan: string;
	} | null;
	prescription?: {
		content: string;
		indications: string;
	} | null;
}

export const getMedicalRecords = async (
	patientId: number,
): Promise<ServiceResponse<MedicalRecord[]>> => {
	return clinicalService.get<MedicalRecord[]>(
		`/clinical/records/patient/${patientId}`,
		{ notifyError: true },
	);
};

export const getMedicalRecordById = async (
	id: number,
): Promise<ServiceResponse<MedicalRecord>> => {
	return clinicalService.get<MedicalRecord>(`/clinical/records/${id}`, {
		notifyError: true,
	});
};

export const updateCritical = async (
	patientId: number,
): Promise<ServiceResponse<Patient>> => {
	return clinicalService.put<Patient>(
		`/clinical/patients/${patientId}/critical`,
		{},
		{ notifyError: true },
	);
};

export const updateCriticalRevert = async (
	patientId: number,
): Promise<ServiceResponse<Patient>> => {
	return clinicalService.put<Patient>(
		`/clinical/patients/${patientId}/critical-revert`,
		{},
		{ notifyError: true },
	);
};

export type CreateMedicalRecordPayload = {
	patient_id: number;
	date: string;
	motive: string;
	observation: string;
	next_appointment_date?: string;
	next_appointment_start_time?: string;
	next_appointment_end_time?: string;
	soap_record: {
		subjective: string;
		objective: string;
		assessment: string;
		plan: string;
	};
	prescription?: {
		content: string;
		indications: string;
	};
};

export const createMedicalRecord = async (
	payload: CreateMedicalRecordPayload,
): Promise<ServiceResponse<MedicalRecord>> => {
	return clinicalService.post<MedicalRecord>("/clinical/records", payload, {
		notifySuccess: true,
		notifyError: true,
	});
};

export type UpdateMedicalRecordPayload = {
	date?: string;
	motive?: string;
	observation?: string;
	next_appointment_date?: string;
	next_appointment_start_time?: string;
	next_appointment_end_time?: string;
	soap_record?: {
		subjective: string;
		objective: string;
		assessment: string;
		plan: string;
	};
};

export const updateMedicalRecord = async (
	id: number,
	payload: UpdateMedicalRecordPayload,
): Promise<ServiceResponse<MedicalRecord>> => {
	return clinicalService.put<MedicalRecord>(
		`/clinical/records/${id}`,
		payload,
		{
			notifySuccess: true,
			notifyError: true,
		},
	);
};

export type UpdatePrescriptionPayload = {
	content: string;
	indications: string;
};

export const updatePrescription = async (
	recordId: number,
	payload: UpdatePrescriptionPayload,
): Promise<ServiceResponse<MedicalRecord>> => {
	return clinicalService.put<MedicalRecord>(
		`/clinical/records/${recordId}/prescription`,
		payload,
		{
			notifySuccess: true,
			notifyError: true,
		},
	);
};
