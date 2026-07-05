import { useDraggable } from "@dnd-kit/core";
import { CSS } from "@dnd-kit/utilities";
import type { Appointment } from "@/api/clinical-service";
import { cn } from "@/lib/utils";
import { getEventColor, getEventPosition } from "./appointment-utils";

interface AppointmentBlockProps {
	appointment: Appointment;
	onClick: () => void;
}

export function AppointmentBlock({
	appointment,
	onClick,
}: AppointmentBlockProps) {
	const { attributes, listeners, setNodeRef, transform, isDragging } =
		useDraggable({
			id: String(appointment.ID),
			disabled:
				appointment.status === "cancelled" ||
				appointment.status === "completed",
		});

	const pos = getEventPosition(appointment.start_time, appointment.end_time);
	const color = getEventColor(appointment.color_id);
	const patientName = appointment.patient
		? appointment.patient.full_name ||
			`${appointment.patient.first_name} ${appointment.patient.last_name}`
		: "";

	return (
		<button
			type="button"
			ref={setNodeRef}
			className={cn(
				"absolute left-1 right-1 rounded-md border-l-[3px] px-2 py-1 cursor-pointer transition-all hover:shadow-md hover:scale-[1.02] overflow-hidden text-left z-10",
				color.bg,
				color.border,
				color.text,
				appointment.status === "cancelled" && "opacity-50 line-through",
				isDragging && "z-30 shadow-lg opacity-80 cursor-grabbing",
			)}
			style={{
				top: `${pos.top}px`,
				height: `${pos.height}px`,
				transform: CSS.Translate.toString(transform),
				transition: isDragging ? "none" : undefined,
			}}
			{...listeners}
			{...attributes}
			onClick={(e) => {
				e.stopPropagation();
				if (isDragging) return;
				onClick();
			}}
		>
			<p className="text-xs font-semibold truncate leading-tight">
				{appointment.title}
			</p>
			{pos.height > 36 && (
				<p className="text-[10px] opacity-80 truncate">
					{appointment.start_time} - {appointment.end_time}
				</p>
			)}
			{pos.height > 52 && (
				<p className="text-[10px] opacity-70 truncate">{patientName}</p>
			)}
		</button>
	);
}

interface AppointmentGhostProps {
	appointment: Appointment;
	startTime: string;
	endTime: string;
}

// Non-interactive duplicate rendered at the snapped drop target while
// dragging, so the user can see exactly where/when the appointment will land.
export function AppointmentGhost({
	appointment,
	startTime,
	endTime,
}: AppointmentGhostProps) {
	const pos = getEventPosition(startTime, endTime);
	const color = getEventColor(appointment.color_id);

	return (
		<div
			className={cn(
				"absolute left-1 right-1 rounded-md border-2 border-dashed px-2 py-1 pointer-events-none z-20 overflow-hidden opacity-25",
				color.bg,
				color.border,
			)}
			style={{
				top: `${pos.top}px`,
				height: `${pos.height}px`,
			}}
		>
			<p
				className={cn(
					"text-xs font-semibold truncate leading-tight",
					color.text,
				)}
			>
				{appointment.title}
			</p>
			<p className={cn("text-[10px] font-medium truncate", color.text)}>
				{startTime} - {endTime}
			</p>
		</div>
	);
}
