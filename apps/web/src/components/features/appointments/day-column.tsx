import { useDroppable } from "@dnd-kit/core";
import { format, isToday } from "date-fns";
import type { Appointment } from "@/api/clinical-service";
import type { ShadeRange } from "@/components/features/agenda/agenda-utils";
import { cn } from "@/lib/utils";
import { AppointmentBlock, AppointmentGhost } from "./appointment-block";
import { HOUR_HEIGHT, START_HOUR } from "./appointment-utils";
import { CurrentTimeLine } from "./current-time-line";

/** Diagonal stripes that mark blocked time. */
const BLOCK_STRIPES =
	"repeating-linear-gradient(135deg, color-mix(in oklab, var(--color-muted-foreground) 18%, transparent) 0 6px, transparent 6px 12px)";

export interface DragGhost {
	/** The column it would land in (see `DayColumn.columnKey`). */
	columnKey: string;
	appointment: Appointment;
	startTime: string;
	endTime: string;
}

interface DayColumnProps {
	day: Date;
	/** Drop target id; defaults to the day ("yyyy-MM-dd"). */
	columnKey?: string;
	hours: number[];
	appointments: Appointment[];
	ghost?: DragGhost | null;
	/** Time outside the doctor's hours and blocked time, drawn behind the slots. */
	shading?: ShadeRange[];
	className?: string;
	/** Absent when the user can't schedule: the slots are then inert. */
	onSlotClick?: (day: Date, hour: number) => void;
	onAppointmentClick: (appointment: Appointment) => void;
	/** Whether appointments can be dragged to another slot. */
	canReschedule?: boolean;
}

export function DayColumn({
	day,
	columnKey,
	hours,
	appointments,
	ghost,
	shading,
	className,
	onSlotClick,
	onAppointmentClick,
	canReschedule = true,
}: DayColumnProps) {
	const { setNodeRef } = useDroppable({
		id: columnKey ?? format(day, "yyyy-MM-dd"),
	});
	const today = isToday(day);

	return (
		<div
			ref={setNodeRef}
			className={cn(
				"relative border-r last:border-r-0",
				today && "bg-primary/2",
				className,
			)}
		>
			{/* Out-of-hours and blocked time (behind the transparent slots) */}
			{shading?.map((shade) => (
				<div
					key={`${shade.kind}-${shade.start}-${shade.end}`}
					aria-hidden
					className={cn(
						"pointer-events-none absolute inset-x-0 overflow-hidden",
						shade.kind === "outside" ? "bg-muted/70" : "bg-muted/40",
					)}
					style={{
						top: `${((shade.start - START_HOUR * 60) / 60) * HOUR_HEIGHT}px`,
						height: `${((shade.end - shade.start) / 60) * HOUR_HEIGHT}px`,
						...(shade.kind === "blocked" && { backgroundImage: BLOCK_STRIPES }),
					}}
				>
					{shade.kind === "blocked" && shade.reason && (
						<span className="block truncate px-1.5 pt-0.5 text-[11px] text-muted-foreground">
							{shade.reason}
						</span>
					)}
				</div>
			))}

			{/* Hour lines (clickable slots) */}
			{hours.map((hour) => (
				<button
					type="button"
					key={hour}
					disabled={!onSlotClick}
					className={cn(
						"absolute w-full border-t border-border/50",
						onSlotClick &&
							"cursor-pointer hover:bg-primary/5 transition-colors",
					)}
					style={{
						top: `${(hour - START_HOUR) * HOUR_HEIGHT}px`,
						height: `${HOUR_HEIGHT}px`,
					}}
					onClick={() => onSlotClick?.(day, hour)}
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
			{appointments.map((appt, index) => (
				<AppointmentBlock
					key={appt.ID}
					index={index}
					appointment={appt}
					draggable={canReschedule}
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
