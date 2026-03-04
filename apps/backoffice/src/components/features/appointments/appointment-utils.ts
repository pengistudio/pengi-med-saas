import { z } from "zod";

// ─── Calendar Constants ──────────────────────────────────────────────────────

export const START_HOUR = 7;
export const END_HOUR = 20;
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
