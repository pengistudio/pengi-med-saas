import {
	DndContext,
	type DragEndEvent,
	type DragMoveEvent,
	PointerSensor,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import { Button, Text } from "@pengi/ui";
import {
	addWeeks,
	eachDayOfInterval,
	endOfWeek,
	format,
	isSameDay,
	isToday,
	startOfWeek,
	subWeeks,
} from "date-fns";
import { es } from "date-fns/locale";
import { ChevronLeft, ChevronRight, Plus } from "lucide-react";
import React from "react";
import {
	type Appointment,
	getAppointments,
	updateAppointment,
} from "@/api/clinical-service";
import { cn } from "@/lib/utils";
import { AppointmentDetailDialog } from "./appointment-detail-dialog";
import { AppointmentFormDialog } from "./appointment-form-dialog";
import {
	END_HOUR,
	HOUR_HEIGHT,
	minutesToTime,
	START_HOUR,
	timeToMinutes,
} from "./appointment-utils";
import { DayColumn, type DragGhost } from "./day-column";

// Given a dragged appointment and the current drop target, computes the
// snapped (15-min) destination day/time — shared by the live ghost preview
// and the final persisted update so both agree on the exact same slot.
function computeSnappedTarget(
	appt: Appointment,
	over: { id: string | number } | null | undefined,
	delta: { x: number; y: number },
	weekDays: Date[],
): { day: Date; startTime: string; endTime: string } | null {
	if (!over) return null;
	const targetDay = weekDays.find(
		(d) => format(d, "yyyy-MM-dd") === String(over.id),
	);
	if (!targetDay) return null;

	const duration =
		timeToMinutes(appt.end_time) - timeToMinutes(appt.start_time);
	const deltaMinutes = delta.y / (HOUR_HEIGHT / 60);
	let newStartMinutes =
		Math.round((timeToMinutes(appt.start_time) + deltaMinutes) / 15) * 15;
	newStartMinutes = Math.min(
		Math.max(newStartMinutes, START_HOUR * 60),
		END_HOUR * 60 - duration,
	);

	return {
		day: targetDay,
		startTime: minutesToTime(newStartMinutes),
		endTime: minutesToTime(newStartMinutes + duration),
	};
}

export default function AppointmentCalendar() {
	const [currentDate, setCurrentDate] = React.useState(new Date());
	const [appointments, setAppointments] = React.useState<Appointment[]>([]);
	const [selectedAppointment, setSelectedAppointment] =
		React.useState<Appointment | null>(null);
	const [showDetailDialog, setShowDetailDialog] = React.useState(false);
	const [showFormDialog, setShowFormDialog] = React.useState(false);
	const [editTarget, setEditTarget] = React.useState<Appointment | null>(null);
	const [createDefaults, setCreateDefaults] = React.useState<{
		date?: Date;
		time?: string;
	}>({});

	const weekStart = React.useMemo(
		() => startOfWeek(currentDate, { weekStartsOn: 1 }),
		[currentDate],
	);
	const weekEnd = React.useMemo(
		() => endOfWeek(currentDate, { weekStartsOn: 1 }),
		[currentDate],
	);
	const weekDays = eachDayOfInterval({ start: weekStart, end: weekEnd });
	const hours = Array.from(
		{ length: END_HOUR - START_HOUR },
		(_, i) => START_HOUR + i,
	);

	const fetchAppointments = React.useCallback(() => {
		const start = format(weekStart, "yyyy-MM-dd");
		const end = format(weekEnd, "yyyy-MM-dd");
		getAppointments(start, end).then((res) => {
			if (res.success && res.data) {
				setAppointments(res.data as Appointment[]);
			}
		});
	}, [weekEnd, weekStart]);

	React.useEffect(() => {
		fetchAppointments();
	}, [fetchAppointments]);

	const [dragGhost, setDragGhost] = React.useState<DragGhost | null>(null);

	const sensors = useSensors(
		useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
	);

	function getAppointmentsForDay(day: Date) {
		return appointments.filter((a) => isSameDay(new Date(a.date), day));
	}

	function handleDragMove(event: DragMoveEvent) {
		const { active, over, delta } = event;
		const appt = appointments.find((a) => a.ID === Number(active.id));
		if (!appt) {
			setDragGhost(null);
			return;
		}

		const target = computeSnappedTarget(appt, over, delta, weekDays);
		if (!target) {
			setDragGhost(null);
			return;
		}

		setDragGhost({
			dayKey: format(target.day, "yyyy-MM-dd"),
			appointment: appt,
			startTime: target.startTime,
			endTime: target.endTime,
		});
	}

	function handleDragCancel() {
		setDragGhost(null);
	}

	function handleDragEnd(event: DragEndEvent) {
		setDragGhost(null);
		const { active, over, delta } = event;
		if (!over) return;

		const apptId = Number(active.id);
		const appt = appointments.find((a) => a.ID === apptId);
		if (!appt) return;

		const target = computeSnappedTarget(appt, over, delta, weekDays);
		if (!target) return;
		const {
			day: targetDay,
			startTime: newStartTime,
			endTime: newEndTime,
		} = target;
		const newDate = targetDay.toISOString();

		if (
			isSameDay(new Date(appt.date), targetDay) &&
			appt.start_time === newStartTime &&
			appt.end_time === newEndTime
		) {
			return;
		}

		const previous = { ...appt };
		setAppointments((prev) =>
			prev.map((a) =>
				a.ID === apptId
					? {
							...a,
							date: newDate,
							start_time: newStartTime,
							end_time: newEndTime,
						}
					: a,
			),
		);

		updateAppointment(apptId, {
			date: newDate,
			start_time: newStartTime,
			end_time: newEndTime,
		}).then((res) => {
			if (!res.success) {
				setAppointments((prev) =>
					prev.map((a) => (a.ID === apptId ? previous : a)),
				);
			} else {
				fetchAppointments();
			}
		});
	}

	function handleSlotClick(day: Date, hour: number) {
		setEditTarget(null);
		setCreateDefaults({
			date: day,
			time: `${hour.toString().padStart(2, "0")}:00`,
		});
		setShowFormDialog(true);
	}

	function handleEditAppointment(appt: Appointment) {
		setEditTarget(appt);
		setCreateDefaults({});
		setShowFormDialog(true);
	}

	function handleNewAppointment() {
		setEditTarget(null);
		setCreateDefaults({ date: new Date() });
		setShowFormDialog(true);
	}

	return (
		<div className="flex flex-col max-h-[calc(100vh-8rem)]">
			{/* ── Header ─────────────────────────────────── */}
			<div className="flex items-center justify-between pb-4 gap-4 flex-wrap">
				<div className="flex items-center gap-4">
					<h1 className="text-2xl font-bold tracking-tight">
						<Text uuid="appointments.title" />
					</h1>
					<div className="flex items-center gap-1">
						<Button
							variant="outline"
							size="icon"
							onClick={() => setCurrentDate(subWeeks(currentDate, 1))}
						>
							<ChevronLeft className="h-4 w-4" />
						</Button>
						<Button
							variant="outline"
							onClick={() => setCurrentDate(new Date())}
							className="px-4"
						>
							<Text uuid="appointments.today" />
						</Button>
						<Button
							variant="outline"
							size="icon"
							onClick={() => setCurrentDate(addWeeks(currentDate, 1))}
						>
							<ChevronRight className="h-4 w-4" />
						</Button>
					</div>
				</div>
				<div className="flex items-center gap-3">
					<p className="text-lg font-medium text-muted-foreground capitalize">
						{format(weekStart, "d MMM", { locale: es })} —{" "}
						{format(weekEnd, "d MMM yyyy", { locale: es })}
					</p>
					<Button onClick={handleNewAppointment}>
						<Plus className="mr-2 h-4 w-4" />
						<Text uuid="appointments.new" />
					</Button>
				</div>
			</div>

			{/* ── Calendar Grid ──────────────────────────── */}
			<div className="flex-1 border rounded-xl overflow-hidden bg-card">
				{/* Scrollable area with headers inside */}
				<div
					className="overflow-auto flex-1 max-h-[calc(100vh-16rem)]"
					style={{ scrollbarGutter: "stable" }}
				>
					{/* Day Headers (sticky) */}
					<div className="grid grid-cols-[60px_repeat(7,1fr)] border-b bg-card sticky top-0 z-20">
						<div className="border-r" />
						{weekDays.map((day) => (
							<div
								key={day.toISOString()}
								className={cn(
									"text-center py-3 border-r last:border-r-0",
									isToday(day) && "bg-primary/5",
								)}
							>
								<p className="text-xs font-medium text-muted-foreground uppercase">
									{format(day, "EEE", { locale: es })}
								</p>
								<p
									className={cn(
										"text-lg font-semibold mt-0.5 leading-none",
										isToday(day) &&
											"bg-primary text-primary-foreground rounded-full w-8 h-8 flex items-center justify-center mx-auto",
									)}
								>
									{format(day, "d")}
								</p>
							</div>
						))}
					</div>

					{/* Time Grid */}
					<DndContext
						sensors={sensors}
						onDragMove={handleDragMove}
						onDragEnd={handleDragEnd}
						onDragCancel={handleDragCancel}
					>
						<div
							className="grid grid-cols-[60px_repeat(7,1fr)] relative"
							style={{
								height: `${hours.length * HOUR_HEIGHT}px`,
							}}
						>
							{/* Time Labels */}
							<div className="relative border-r">
								{hours.map((hour) => (
									<div
										key={hour}
										className="absolute w-full pr-2 text-right"
										style={{
											top: `${(hour - START_HOUR) * HOUR_HEIGHT}px`,
										}}
									>
										<span className="text-xs text-muted-foreground block">
											{`${hour.toString().padStart(2, "0")}:00`}
										</span>
									</div>
								))}
							</div>

							{/* Day Columns */}
							{weekDays.map((day) => {
								const dayKey = format(day, "yyyy-MM-dd");
								return (
									<DayColumn
										key={day.toISOString()}
										day={day}
										hours={hours}
										appointments={getAppointmentsForDay(day)}
										ghost={dragGhost?.dayKey === dayKey ? dragGhost : null}
										onSlotClick={handleSlotClick}
										onAppointmentClick={(appt) => {
											setSelectedAppointment(appt);
											setShowDetailDialog(true);
										}}
									/>
								);
							})}
						</div>
					</DndContext>
				</div>
			</div>

			{/* ── Dialogs ─────────────────────────────────── */}
			<AppointmentDetailDialog
				appointment={selectedAppointment}
				open={showDetailDialog}
				onOpenChange={setShowDetailDialog}
				onEdit={handleEditAppointment}
				onRefresh={fetchAppointments}
			/>

			<AppointmentFormDialog
				open={showFormDialog}
				onOpenChange={setShowFormDialog}
				appointment={editTarget}
				defaultDate={createDefaults.date}
				defaultTime={createDefaults.time}
				onSuccess={fetchAppointments}
			/>
		</div>
	);
}
