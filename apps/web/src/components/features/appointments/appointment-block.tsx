import { useDraggable } from "@dnd-kit/core";
import { CSS } from "@dnd-kit/utilities";
import React from "react";
import type { Appointment } from "@/api/clinical-service";
import { cn } from "@/lib/utils";
import { clearLanding, peekLanding } from "./appointment-landing";
import { getAppointmentColor, getEventPosition } from "./appointment-utils";

interface AppointmentBlockProps {
	appointment: Appointment;
	onClick: () => void;
	index?: number;
	draggable?: boolean;
}

export function AppointmentBlock({
	appointment,
	onClick,
	index = 0,
	draggable = true,
}: AppointmentBlockProps) {
	const { attributes, listeners, setNodeRef, transform, isDragging } =
		useDraggable({
			id: String(appointment.ID),
			disabled:
				!draggable ||
				appointment.status === "cancelled" ||
				appointment.status === "completed",
		});

	// Dropped on another day, the block remounts in that column: start it where
	// it was released and slide it into its slot instead of popping in.
	const [landing] = React.useState(() => peekLanding(appointment.ID));
	const ref = React.useRef<HTMLButtonElement>(null);
	React.useLayoutEffect(() => {
		if (!landing || !ref.current) return;
		clearLanding(appointment.ID);
		const to = ref.current.getBoundingClientRect();
		ref.current.animate(
			[
				{ translate: `${landing.left - to.left}px ${landing.top - to.top}px` },
				{ translate: "0 0" },
			],
			{ duration: 220, easing: "cubic-bezier(0.22, 1, 0.36, 1)" },
		);
	}, [landing, appointment.ID]);

	const pos = getEventPosition(appointment.start_time, appointment.end_time);
	const color = getAppointmentColor(appointment);
	const patientName = appointment.patient
		? appointment.patient.full_name ||
			`${appointment.patient.first_name} ${appointment.patient.last_name}`
		: "";

	return (
		<button
			type="button"
			ref={(el) => {
				setNodeRef(el);
				ref.current = el;
			}}
			className={cn(
				"absolute left-1 right-1 rounded-md border-l-[3px] px-2 py-1 cursor-pointer transition-all duration-(--motion-base) ease-out-soft hover:shadow-md hover:scale-[1.02] overflow-hidden text-left z-10",
				!landing &&
					"animate-in fade-in-0 zoom-in-95 [animation-fill-mode:backwards]",
				color.className,
				appointment.status === "cancelled" && "opacity-50 line-through",
				isDragging && "z-30 shadow-lg opacity-80 cursor-grabbing",
			)}
			style={{
				...color.style,
				top: `${pos.top}px`,
				height: `${pos.height}px`,
				transform: CSS.Translate.toString(transform),
				transition: isDragging ? "none" : undefined,
				animationDelay: `${Math.min(index, 5) * 30}ms`,
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
			{pos.height > 68 && appointment.doctor && (
				<p className="text-[10px] opacity-70 truncate">
					{appointment.doctor.full_name}
				</p>
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
	const color = getAppointmentColor(appointment);

	return (
		<div
			className={cn(
				"absolute left-1 right-1 rounded-md border-2 border-dashed px-2 py-1 pointer-events-none z-20 overflow-hidden opacity-25",
				color.className,
			)}
			style={{
				...color.style,
				top: `${pos.top}px`,
				height: `${pos.height}px`,
			}}
		>
			<p
				className={cn(
					"text-xs font-semibold truncate leading-tight",
					color.textClassName,
				)}
			>
				{appointment.title}
			</p>
			<p
				className={cn("text-[10px] font-medium truncate", color.textClassName)}
			>
				{startTime} - {endTime}
			</p>
		</div>
	);
}
