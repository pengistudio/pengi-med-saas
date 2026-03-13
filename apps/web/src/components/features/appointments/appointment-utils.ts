import { z } from "zod";

// ─── Calendar Constants ──────────────────────────────────────────────────────

export const START_HOUR = 6;
export const END_HOUR = 24;
export const HOUR_HEIGHT = 64;

// ─── Status Colors ───────────────────────────────────────────────────────────

export const STATUS_COLORS = {
	scheduled: {
		bg: "bg-blue-500/15",
		border: "border-blue-500",
		text: "text-blue-700 dark:text-blue-400",
		dot: "bg-blue-500",
		badge: "bg-blue-500/20 text-blue-700 dark:text-blue-400",
	},
	arrived: {
		bg: "bg-amber-500/15",
		border: "border-amber-500",
		text: "text-amber-700 dark:text-amber-400",
		dot: "bg-amber-500",
		badge: "bg-amber-500/20 text-amber-700 dark:text-amber-400",
	},
	in_consultation: {
		bg: "bg-violet-500/15",
		border: "border-violet-500",
		text: "text-violet-700 dark:text-violet-400",
		dot: "bg-violet-500",
		badge: "bg-violet-500/20 text-violet-700 dark:text-violet-400",
	},
	completed: {
		bg: "bg-emerald-500/15",
		border: "border-emerald-500",
		text: "text-emerald-700 dark:text-emerald-400",
		dot: "bg-emerald-500",
		badge: "bg-emerald-500/20 text-emerald-700 dark:text-emerald-400",
	},
	cancelled: {
		bg: "bg-gray-400/15",
		border: "border-gray-400",
		text: "text-gray-500 dark:text-gray-400",
		dot: "bg-gray-400",
		badge: "bg-gray-400/20 text-gray-500 dark:text-gray-400",
	},
};

export type StatusColorKey = keyof typeof STATUS_COLORS;

export function getStatusColor(status: string) {
	return STATUS_COLORS[status as StatusColorKey] || STATUS_COLORS.scheduled;
}

// ─── Status i18n Keys ────────────────────────────────────────────────────────

export const STATUS_I18N_KEYS: Record<string, string> = {
	scheduled: "appointments.status.scheduled",
	arrived: "appointments.status.arrived",
	in_consultation: "appointments.status.in_consultation",
	completed: "appointments.status.completed",
	cancelled: "appointments.status.cancelled",
};

// ─── Helpers ─────────────────────────────────────────────────────────────────

export function timeToMinutes(time: string): number {
	const [h, m] = time.split(":").map(Number);
	return h * 60 + m;
}

export function getEventPosition(startTime: string, endTime: string) {
	const startMinutes = timeToMinutes(startTime) - START_HOUR * 60;
	const endMinutes = timeToMinutes(endTime) - START_HOUR * 60;
	const top = (startMinutes / 60) * HOUR_HEIGHT;
	const height = Math.max(((endMinutes - startMinutes) / 60) * HOUR_HEIGHT, 24);
	return { top, height };
}

// ─── Appointment Form Schema ─────────────────────────────────────────────────

export const appointmentSchema = z.object({
	title: z.string().min(1, "Requerido"),
	start_time: z.string().min(1, "Requerido"),
	end_time: z.string().min(1, "Requerido"),
	location: z.string().optional(),
	notes: z.string().optional(),
});
