import {
	type BaseModel,
	createHttpService,
	type ServiceResponse,
} from "@pengi/shared";
import { apiWithTenant } from ".";

// Doctor schedules, schedule blocks, appointment types and the agenda reads
// built on them (availability warning, shading). Times are "HH:MM" local
// (an end may be "24:00"), dates "YYYY-MM-DD".

const agendaService = createHttpService(apiWithTenant);

/** One weekly working range of a doctor. */
export interface DoctorSchedule extends BaseModel {
	tenant_id: number;
	doctor_id: number;
	/** 0 = Sunday … 6 = Saturday. */
	weekday: number;
	start_time: string;
	end_time: string;
}

export type ScheduleSlot = Pick<
	DoctorSchedule,
	"weekday" | "start_time" | "end_time"
>;

/** An exception to the schedule: days (or a time range on those days) off. */
export interface ScheduleBlock extends BaseModel {
	tenant_id: number;
	/** null: the whole clinic. */
	doctor_id: number | null;
	start_date: string;
	end_date: string;
	/** Both empty: full days. */
	start_time: string;
	end_time: string;
	reason: string;
}

export type BlockPayload = Pick<
	ScheduleBlock,
	"start_date" | "end_date" | "start_time" | "end_time" | "reason"
>;

/** An appointment that falls inside a block just created or changed. */
export interface AffectedAppointment {
	id: number;
	/** Same as Appointment.date. */
	date: string;
	start_time: string;
	end_time: string;
	status: string;
	doctor_id: number | null;
	patient_id: number;
	patient_name: string;
}

export interface BlockResponse {
	block: ScheduleBlock;
	affected_appointments: AffectedAppointment[];
}

export interface AppointmentType extends BaseModel {
	tenant_id: number;
	name: string;
	duration_minutes: number;
	/** #RRGGBB or "". */
	color: string;
	active: boolean;
	/** Empty: every doctor. */
	doctor_ids: number[];
}

export type AppointmentTypePayload = {
	name: string;
	duration_minutes: number;
	color: string;
	active: boolean;
	doctor_ids: number[];
};

export interface TimeRange {
	start_time: string;
	end_time: string;
}

export type AvailabilityStatus =
	| "inside"
	| "outside"
	| "blocked"
	| "no_schedule";

export interface Availability {
	status: AvailabilityStatus;
	/** The doctor's merged working ranges that weekday. */
	ranges: TimeRange[];
	/** Blocks overlapping the slot. */
	blocks: ScheduleBlock[];
}

export interface AgendaDay {
	date: string;
	ranges: TimeRange[];
	/** The doctor's and the clinic's blocks of that day. */
	blocks: ScheduleBlock[];
}

export interface DoctorAgenda {
	doctor_id: number;
	active: boolean;
	/** false: never shade this doctor as "outside" (no schedule at all). */
	has_schedule: boolean;
	days: AgendaDay[];
}

export interface AgendaRange {
	from: string;
	to: string;
	doctors: DoctorAgenda[];
	/** Clinic-wide blocks per day, for the "no doctor" column. */
	clinic: { date: string; blocks: ScheduleBlock[] }[];
}

/** Whose schedule: the current user's own doctor profile, or a doctor's ID. */
export type ScheduleScope = "me" | number;

/** Whose blocks: own, a doctor's, or the whole clinic's. */
export type BlockScope = "me" | "clinic" | number;

const schedulePath = (scope: ScheduleScope) => `/doctors/${scope}/schedule`;

function blocksPath(scope: BlockScope) {
	return scope === "clinic" ? "/agenda/blocks" : `/doctors/${scope}/blocks`;
}

// ─── Weekly schedule ─────────────────────────────────────────────────────────

export const getSchedule = async (
	scope: ScheduleScope,
): Promise<ServiceResponse<DoctorSchedule[]>> =>
	agendaService.get<DoctorSchedule[]>(schedulePath(scope), {
		notifyError: true,
	});

/** Replaces the whole week; an empty list clears it. */
export const replaceSchedule = async (
	scope: ScheduleScope,
	slots: ScheduleSlot[],
): Promise<ServiceResponse<DoctorSchedule[]>> =>
	agendaService.put<DoctorSchedule[]>(
		schedulePath(scope),
		{ slots },
		{ notifySuccess: true, notifyError: true },
	);

// ─── Blocks ──────────────────────────────────────────────────────────────────

/** Blocks ending on or after `from` (default today); `to` bounds the start. */
export const getBlocks = async (
	scope: BlockScope,
	params: { from?: string; to?: string } = {},
): Promise<ServiceResponse<ScheduleBlock[]>> => {
	const qs = new URLSearchParams();
	if (params.from) qs.set("from", params.from);
	if (params.to) qs.set("to", params.to);
	const query = qs.toString() ? `?${qs.toString()}` : "";
	return agendaService.get<ScheduleBlock[]>(`${blocksPath(scope)}${query}`, {
		notifyError: true,
	});
};

export const createBlock = async (
	scope: BlockScope,
	payload: BlockPayload,
): Promise<ServiceResponse<BlockResponse>> =>
	agendaService.post<BlockResponse>(blocksPath(scope), payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateBlock = async (
	scope: BlockScope,
	id: number,
	payload: BlockPayload,
): Promise<ServiceResponse<BlockResponse>> =>
	agendaService.put<BlockResponse>(`${blocksPath(scope)}/${id}`, payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const deleteBlock = async (
	scope: BlockScope,
	id: number,
): Promise<ServiceResponse<null>> =>
	agendaService.delete<null>(`${blocksPath(scope)}/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});

// ─── Appointment types ───────────────────────────────────────────────────────

/**
 * Every type of the tenant, active or not, sorted by name. Silent on error:
 * it also feeds the appointment form in the background.
 */
export const getAppointmentTypes = async (): Promise<
	ServiceResponse<AppointmentType[]>
> =>
	agendaService.get<AppointmentType[]>("/agenda/appointment-types", {
		notifyError: false,
	});

export const createAppointmentType = async (
	payload: AppointmentTypePayload,
): Promise<ServiceResponse<AppointmentType>> =>
	agendaService.post<AppointmentType>("/agenda/appointment-types", payload, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateAppointmentType = async (
	id: number,
	payload: Partial<AppointmentTypePayload>,
): Promise<ServiceResponse<AppointmentType>> =>
	agendaService.put<AppointmentType>(
		`/agenda/appointment-types/${id}`,
		payload,
		{ notifySuccess: true, notifyError: true },
	);

export const deleteAppointmentType = async (
	id: number,
): Promise<ServiceResponse<null>> =>
	agendaService.delete<null>(`/agenda/appointment-types/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});

// ─── Agenda reads (background, no toasts) ────────────────────────────────────

export const getAvailability = async (params: {
	doctor_id: number;
	date: string;
	start_time: string;
	end_time: string;
}): Promise<ServiceResponse<Availability>> => {
	const qs = new URLSearchParams({
		doctor_id: String(params.doctor_id),
		date: params.date,
		start_time: params.start_time,
		end_time: params.end_time,
	});
	return agendaService.get<Availability>(
		`/agenda/availability?${qs.toString()}`,
		{ notifyError: false },
	);
};

/** Working ranges and blocks per day (max 62 days); no ids: active doctors. */
export const getAgendaRange = async (
	from: string,
	to: string,
	doctorIds?: number[],
): Promise<ServiceResponse<AgendaRange>> => {
	const qs = new URLSearchParams({ from, to });
	if (doctorIds?.length) qs.set("doctor_ids", doctorIds.join(","));
	return agendaService.get<AgendaRange>(`/agenda/range?${qs.toString()}`, {
		notifyError: false,
	});
};
