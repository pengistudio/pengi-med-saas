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
import { type Appointment, getAppointments } from "@/api/clinical-service";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { AppointmentDetailDialog } from "./appointment-detail-dialog";
import { AppointmentFormDialog } from "./appointment-form-dialog";
import {
	END_HOUR,
	getEventPosition,
	getStatusColor,
	HOUR_HEIGHT,
	START_HOUR,
} from "./appointment-utils";
import { CurrentTimeLine } from "./current-time-line";

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

	function getAppointmentsForDay(day: Date) {
		return appointments.filter((a) => isSameDay(new Date(a.date), day));
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
		<div className="flex flex-col h-[calc(100vh-8rem)]">
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
				{/* Day Headers */}
				<div className="grid grid-cols-[60px_repeat(7,1fr)] border-b bg-muted/30 sticky top-0 z-10">
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
				<div className="overflow-auto flex-1 max-h-[calc(100vh-16rem)]">
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
									<span className="text-xs text-muted-foreground -mt-2 block">
										{`${hour.toString().padStart(2, "0")}:00`}
									</span>
								</div>
							))}
						</div>

						{/* Day Columns */}
						{weekDays.map((day) => {
							const dayAppointments = getAppointmentsForDay(day);
							return (
								<div
									key={day.toISOString()}
									className={cn(
										"relative border-r last:border-r-0",
										isToday(day) && "bg-primary/2",
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
											onClick={() => handleSlotClick(day, hour)}
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
									{dayAppointments.map((appt) => {
										const pos = getEventPosition(
											appt.start_time,
											appt.end_time,
										);
										const color = getStatusColor(appt.status);
										const patientName = appt.patient
											? appt.patient.full_name ||
												`${appt.patient.first_name} ${appt.patient.last_name}`
											: "";
										return (
											<button
												type="button"
												key={appt.ID}
												className={cn(
													"absolute left-1 right-1 rounded-md border-l-[3px] px-2 py-1 cursor-pointer transition-all hover:shadow-md hover:scale-[1.02] overflow-hidden text-left z-10",
													color.bg,
													color.border,
													color.text,
													appt.status === "cancelled" &&
														"opacity-50 line-through",
												)}
												style={{
													top: `${pos.top}px`,
													height: `${pos.height}px`,
												}}
												onClick={(e) => {
													e.stopPropagation();
													setSelectedAppointment(appt);
													setShowDetailDialog(true);
												}}
											>
												<p className="text-xs font-semibold truncate leading-tight">
													{appt.title}
												</p>
												{pos.height > 36 && (
													<p className="text-[10px] opacity-80 truncate">
														{appt.start_time} - {appt.end_time}
													</p>
												)}
												{pos.height > 52 && (
													<p className="text-[10px] opacity-70 truncate">
														{patientName}
													</p>
												)}
											</button>
										);
									})}

									{/* Current time indicator */}
									{isToday(day) && <CurrentTimeLine />}
								</div>
							);
						})}
					</div>
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
