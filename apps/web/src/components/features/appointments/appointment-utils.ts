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

// ─── Event Colors (manual, synced to Google Calendar) ───────────────────────
// Matches Google Calendar's fixed 11-color event palette 1:1 (by colorId) so
// the app and Google show the exact same color, with no lossy conversion.

export const EVENT_COLORS = [
	{
		id: "1",
		name: "lavender",
		hex: "#7986cb",
		bg: "bg-[#7986cb]/15",
		border: "border-[#7986cb]",
		text: "text-[#7986cb]",
		dot: "bg-[#7986cb]",
	},
	{
		id: "2",
		name: "sage",
		hex: "#33b679",
		bg: "bg-[#33b679]/15",
		border: "border-[#33b679]",
		text: "text-[#33b679]",
		dot: "bg-[#33b679]",
	},
	{
		id: "3",
		name: "grape",
		hex: "#8e24aa",
		bg: "bg-[#8e24aa]/15",
		border: "border-[#8e24aa]",
		text: "text-[#8e24aa]",
		dot: "bg-[#8e24aa]",
	},
	{
		id: "4",
		name: "flamingo",
		hex: "#e67c73",
		bg: "bg-[#e67c73]/15",
		border: "border-[#e67c73]",
		text: "text-[#e67c73]",
		dot: "bg-[#e67c73]",
	},
	{
		id: "5",
		name: "banana",
		hex: "#f6c026",
		bg: "bg-[#f6c026]/15",
		border: "border-[#f6c026]",
		text: "text-[#f6c026]",
		dot: "bg-[#f6c026]",
	},
	{
		id: "6",
		name: "tangerine",
		hex: "#f5511d",
		bg: "bg-[#f5511d]/15",
		border: "border-[#f5511d]",
		text: "text-[#f5511d]",
		dot: "bg-[#f5511d]",
	},
	{
		id: "7",
		name: "peacock",
		hex: "#039be5",
		bg: "bg-[#039be5]/15",
		border: "border-[#039be5]",
		text: "text-[#039be5]",
		dot: "bg-[#039be5]",
	},
	{
		id: "8",
		name: "graphite",
		hex: "#616161",
		bg: "bg-[#616161]/15",
		border: "border-[#616161]",
		text: "text-[#616161]",
		dot: "bg-[#616161]",
	},
	{
		id: "9",
		name: "blueberry",
		hex: "#3f51b5",
		bg: "bg-[#3f51b5]/15",
		border: "border-[#3f51b5]",
		text: "text-[#3f51b5]",
		dot: "bg-[#3f51b5]",
	},
	{
		id: "10",
		name: "basil",
		hex: "#0b8043",
		bg: "bg-[#0b8043]/15",
		border: "border-[#0b8043]",
		text: "text-[#0b8043]",
		dot: "bg-[#0b8043]",
	},
	{
		id: "11",
		name: "tomato",
		hex: "#d60000",
		bg: "bg-[#d60000]/15",
		border: "border-[#d60000]",
		text: "text-[#d60000]",
		dot: "bg-[#d60000]",
	},
] as const;

const DEFAULT_EVENT_COLOR = {
	id: "",
	name: "default",
	hex: "#9ca3af",
	bg: "bg-gray-400/15",
	border: "border-gray-400",
	text: "text-gray-500 dark:text-gray-400",
	dot: "bg-gray-400",
};

export function getEventColor(colorId?: string) {
	if (!colorId) return DEFAULT_EVENT_COLOR;
	return EVENT_COLORS.find((c) => c.id === colorId) || DEFAULT_EVENT_COLOR;
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

export function minutesToTime(minutes: number): string {
	const h = Math.floor(minutes / 60)
		.toString()
		.padStart(2, "0");
	const m = (minutes % 60).toString().padStart(2, "0");
	return `${h}:${m}`;
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
	color_id: z.string().optional(),
});
