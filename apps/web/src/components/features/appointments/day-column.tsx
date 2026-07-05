import { useDroppable } from "@dnd-kit/core";
import { format, isToday } from "date-fns";
import type { Appointment } from "@/api/clinical-service";
import { cn } from "@/lib/utils";
import { AppointmentBlock, AppointmentGhost } from "./appointment-block";
import { HOUR_HEIGHT, START_HOUR } from "./appointment-utils";
import { CurrentTimeLine } from "./current-time-line";

export interface DragGhost {
	dayKey: string;
	appointment: Appointment;
	startTime: string;
	endTime: string;
}

interface DayColumnProps {
	day: Date;
	hours: number[];
	appointments: Appointment[];
	ghost?: DragGhost | null;
	onSlotClick: (day: Date, hour: number) => void;
	onAppointmentClick: (appointment: Appointment) => void;
}

export function DayColumn({
	day,
	hours,
	appointments,
	ghost,
	onSlotClick,
	onAppointmentClick,
}: DayColumnProps) {
	const { setNodeRef } = useDroppable({ id: format(day, "yyyy-MM-dd") });
	const today = isToday(day);

	return (
		<div
			ref={setNodeRef}
			className={cn(
				"relative border-r last:border-r-0",
				today && "bg-primary/2",
			)}
		>
			{/* Hour lines (clickable slots) */}
			{hours.map((hour) => (
				<button
					type="button"
					key={hour}
					className="absolute w-full border-t border-border/50 cursor-pointer hover:bg-primary/5 transition-colors"
					style={{
						top: `${(hour - START_HOUR) * HOUR_HEIGHT}px`,
						height: `${HOUR_HEIGHT}px`,
					}}
					onClick={() => onSlotClick(day, hour)}
				>
					<div
						className="absolute w-full border-t border-border/20"
						style={{
							top: `${HOUR_HEIGHT / 2}px`,
						}}
					/>
				</button>
			))}

			{/* Events */}
			{appointments.map((appt) => (
				<AppointmentBlock
					key={appt.ID}
					appointment={appt}
					onClick={() => onAppointmentClick(appt)}
				/>
			))}

			{/* Drop target preview (ghost) */}
			{ghost && (
				<AppointmentGhost
					appointment={ghost.appointment}
					startTime={ghost.startTime}
					endTime={ghost.endTime}
				/>
			)}

			{/* Current time indicator */}
			{today && <CurrentTimeLine />}
		</div>
	);
}
