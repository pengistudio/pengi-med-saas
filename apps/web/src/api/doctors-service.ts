import {
	type BaseModel,
	createHttpService,
	type ServiceResponse,
} from "@pengi/shared";
import { toast } from "sonner";
import { apiWithTenant } from ".";

const doctorsService = createHttpService(apiWithTenant);

/** A professional who attends patients; independent of the user's role. */
export interface Doctor extends BaseModel {
	tenant_id: number;
	/** Linked account; null for a doctor without one (cannot sign). */
	user_id: number | null;
	full_name: string;
	/** Code from the specialty catalog (GET /doctors/specialties). */
	specialty: string;
	/** Name of the specialty when `specialty` is "other". */
	specialty_other: string;
	id_number: string;
	professional_registry: string;
	phone: string;
	email: string;
	/** Agenda color, #RRGGBB. */
	color: string;
	active: boolean;
	/** Created by the data migration and not reviewed yet. */
	needs_review: boolean;
}

export interface Specialty {
	code: string;
	label_key: string;
}

/** What the frontend needs for onboarding and notices. */
export interface DoctorStatus {
	has_profile: boolean;
	/** The current user's own profile, or null. */
	doctor: Doctor | null;
	active_doctors: number;
	needs_review: boolean;
}

/** Profile fields: what an admin sets and what a linked doctor edits on their own profile. */
export type DoctorProfilePayload = {
	full_name: string;
	specialty: string;
	specialty_other?: string;
	id_number?: string;
	professional_registry?: string;
	phone?: string;
	email?: string;
	/** Empty: the backend assigns the next palette color. */
	color?: string;
};

export type CreateDoctorPayload = DoctorProfilePayload & {
	user_id?: number | null;
};

/** Error code returned when deleting a doctor that documents reference. */
export const DOCTOR_IN_USE_ERROR = "E-DR-007";

/** The specialty "other": its name goes in `specialty_other`. */
export const SPECIALTY_OTHER = "other";

/** Same palette the backend assigns from (doctor_data.Palette). */
export const DOCTOR_COLORS = [
	"#2563EB",
	"#16A34A",
	"#DC2626",
	"#9333EA",
	"#EA580C",
	"#0891B2",
	"#DB2777",
	"#CA8A04",
	"#4F46E5",
	"#0D9488",
	"#65A30D",
	"#7C3AED",
] as const;

// ─── Reads ───────────────────────────────────────────────────────────────────

/**
 * Doctors of the tenant sorted by name; `active` filters them. `notifyError`
 * is off for background loads (selectors) so a user without access to the list
 * doesn't get a toast.
 */
export const getDoctors = async (
	params: { active?: boolean; notifyError?: boolean } = {},
): Promise<ServiceResponse<Doctor[]>> => {
	const qs = params.active === undefined ? "" : `?active=${params.active}`;
	return doctorsService.get<Doctor[]>(`/doctors${qs}`, {
		notifyError: params.notifyError ?? true,
	});
};

export const getDoctorById = async (
	id: number,
): Promise<ServiceResponse<Doctor>> =>
	doctorsService.get<Doctor>(`/doctors/${id}`, { notifyError: true });

export const getSpecialties = async (): Promise<ServiceResponse<Specialty[]>> =>
	doctorsService.get<Specialty[]>("/doctors/specialties", {
		notifyError: false,
	});

/** Onboarding status of the current user; any tenant member may call it. */
export const getDoctorStatus = async (): Promise<
	ServiceResponse<DoctorStatus>
> =>
	doctorsService.get<DoctorStatus>("/doctors/status", { notifyError: false });

// ─── Own profile ─────────────────────────────────────────────────────────────

export const createMyDoctor = async (
	payload: DoctorProfilePayload,
): Promise<ServiceResponse<Doctor>> =>
	doctorsService.post<Doctor>("/doctors/me", payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateMyDoctor = async (
	payload: Partial<DoctorProfilePayload>,
): Promise<ServiceResponse<Doctor>> =>
	doctorsService.put<Doctor>("/doctors/me", payload, {
		notifySuccess: true,
		notifyError: true,
	});

// ─── Administration (MANAGE_DOCTORS) ─────────────────────────────────────────

export const createDoctor = async (
	payload: CreateDoctorPayload,
): Promise<ServiceResponse<Doctor>> =>
	doctorsService.post<Doctor>("/doctors", payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateDoctor = async (
	id: number,
	payload: Partial<DoctorProfilePayload>,
): Promise<ServiceResponse<Doctor>> =>
	doctorsService.put<Doctor>(`/doctors/${id}`, payload, {
		notifySuccess: true,
		notifyError: true,
	});

/** Links the doctor to a user of the tenant, or unlinks it with null. */
export const linkDoctorUser = async (
	id: number,
	userId: number | null,
): Promise<ServiceResponse<Doctor>> =>
	doctorsService.put<Doctor>(
		`/doctors/${id}/user`,
		{ user_id: userId },
		{ notifySuccess: true, notifyError: true },
	);

export const activateDoctor = async (
	id: number,
): Promise<ServiceResponse<Doctor>> =>
	doctorsService.put<Doctor>(
		`/doctors/${id}/activate`,
		{},
		{ notifySuccess: true, notifyError: true },
	);

export const deactivateDoctor = async (
	id: number,
): Promise<ServiceResponse<Doctor>> =>
	doctorsService.put<Doctor>(
		`/doctors/${id}/deactivate`,
		{},
		{ notifySuccess: true, notifyError: true },
	);

/**
 * Hard delete. Fails with 409 `DOCTOR_IN_USE_ERROR` when documents reference
 * the doctor: that one is not toasted, the page offers to deactivate instead;
 * any other error is.
 */
export const deleteDoctor = async (
	id: number,
): Promise<ServiceResponse<null>> => {
	const res = await doctorsService.delete<null>(`/doctors/${id}`, {
		notifySuccess: true,
		notifyError: false,
	});
	if (!res.success && res.data?.error_code !== DOCTOR_IN_USE_ERROR) {
		toast.error(res.message);
	}
	return res;
};

/** Clears every doctor's `needs_review` (the one-time review notice). */
export const markDoctorsReviewed = async (): Promise<ServiceResponse<null>> =>
	doctorsService.put<null>(
		"/doctors/review",
		{},
		{ notifySuccess: true, notifyError: true },
	);
