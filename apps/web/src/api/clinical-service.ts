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
	email: string;
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
	medical_records?: MedicalRecord[];
	appointments?: Appointment[];
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
	email?: string;
	first_name: string;
	last_name: string;
	birth_date?: Date;
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

export interface Appointment extends BaseModel {
	tenant_id: number;
	patient_id: number;
	title: string;
	date: string;
	start_time: string;
	end_time: string;
	location?: string;
	notes?: string;
	status: "scheduled" | "completed" | "cancelled";
	patient?: Patient;
}

export interface MedicalRecord extends BaseModel {
	patient_id: number;
	date: string;
	appointment_id?: number | null;
	appointment?: Appointment | null;
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
	appointment_id?: number;
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
	appointment_id?: number;
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

// ─── Appointment API ─────────────────────────────────────────────────────────

export type CreateAppointmentPayload = {
	patient_id: number;
	title: string;
	date: string;
	start_time: string;
	end_time: string;
	location?: string;
	notes?: string;
};

export type UpdateAppointmentPayload = Partial<CreateAppointmentPayload>;

export const getAppointments = async (
	start?: string,
	end?: string,
): Promise<ServiceResponse<Appointment[]>> => {
	const params = new URLSearchParams();
	if (start) params.set("start", start);
	if (end) params.set("end", end);
	const qs = params.toString() ? `?${params.toString()}` : "";
	return clinicalService.get<Appointment[]>(`/clinical/appointments${qs}`, {
		notifyError: true,
	});
};

export const getPatientAppointments = async (
	patientId: number,
	status?: string,
): Promise<ServiceResponse<Appointment[]>> => {
	const qs = status ? `?status=${status}` : "";
	return clinicalService.get<Appointment[]>(
		`/clinical/appointments/patient/${patientId}${qs}`,
		{ notifyError: true },
	);
};

export const createAppointment = async (
	payload: CreateAppointmentPayload,
): Promise<ServiceResponse<Appointment>> => {
	return clinicalService.post<Appointment>("/clinical/appointments", payload, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const updateAppointment = async (
	id: number,
	payload: UpdateAppointmentPayload,
): Promise<ServiceResponse<Appointment>> => {
	return clinicalService.put<Appointment>(
		`/clinical/appointments/${id}`,
		payload,
		{ notifySuccess: true, notifyError: true },
	);
};

export const updateAppointmentStatus = async (
	id: number,
	status: string,
): Promise<ServiceResponse<Appointment>> => {
	return clinicalService.put<Appointment>(
		`/clinical/appointments/${id}/status`,
		{ status },
		{ notifySuccess: true, notifyError: true },
	);
};

export const deleteAppointment = async (
	id: number,
): Promise<ServiceResponse<null>> => {
	return clinicalService.delete<null>(`/clinical/appointments/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const downloadPrescription = async (
	recordId: number,
): Promise<ServiceResponse<Blob>> => {
	return clinicalService.get<Blob>(
		`/clinical/records/${recordId}/prescription/download`,
		{
			responseType: "blob",
			notifyError: true,
		},
	);
};

// ─── Dashboard API ───────────────────────────────────────────────────────────

export interface WeekDayStat {
	date: string;
	day: string;
	count: number;
}

export interface UpcomingAppointment {
	id: number;
	title: string;
	start_time: string;
	end_time: string;
	patient_name: string;
	patient_id: number;
	status: string;
}

export interface DashboardStats {
	total_patients: number;
	critical_patients: number;
	today_appointments: number;
	monthly_completed: number;
	weekly_appointments: WeekDayStat[];
	upcoming_appointments: UpcomingAppointment[];
}

export const getDashboardStats = async (): Promise<
	ServiceResponse<DashboardStats>
> => {
	return clinicalService.get<DashboardStats>("/clinical/dashboard/stats", {
		notifyError: true,
	});
};
