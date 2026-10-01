import {
	DndContext,
	type DragEndEvent,
	type DragMoveEvent,
} from "@dnd-kit/core";
import { useText } from "@pengi/shared";
import {
	Button,
	Sheet,
	SheetContent,
	SheetTitle,
	Text,
	useViewport,
} from "@pengi/ui";
import {
	addDays,
	addWeeks,
	eachDayOfInterval,
	endOfWeek,
	format,
	isSameDay,
	isToday,
	startOfWeek,
} from "date-fns";
import { es } from "date-fns/locale";
import { CalendarClock, ChevronLeft, ChevronRight, Plus } from "lucide-react";
import React from "react";
import { useSearchParams } from "react-router";
import {
	type Appointment,
	getAppointments,
	type Patient,
	updateAppointment,
} from "@/api/clinical-service";
import { PageHeader } from "@/components/custom/page-header";
import { useDragSensors } from "@/hooks/use-drag-sensors";
import useTenantSettings from "@/hooks/use-tenant-settings";
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
import { PendingFollowUpsPanel } from "./pending-follow-ups-panel";

// Given a dragged appointment and the current drop target, computes the
// snapped (15-min) destination day/time — shared by the live ghost preview
// and the final persisted update so both agree on the exact same slot.
function computeSnappedTarget(
	appt: Appointment,
	over: { id: string | number } | null | undefined,
	delta: { x: number; y: number },
	days: Date[],
): { day: Date; startTime: string; endTime: string } | null {
	if (!over) return null;
	const targetDay = days.find(
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
	const [searchParams] = useSearchParams();
	// ?date=YYYY-MM-DD opens that week (the dashboard's week strip links here).
	const [currentDate, setCurrentDate] = React.useState(() => {
		const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(
			searchParams.get("date") ?? "",
		);
		return match
			? new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]))
			: new Date();
	});
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
	const [suggestedPatient, setSuggestedPatient] =
		React.useState<Patient | null>(null);
	const [pendingRefreshKey, setPendingRefreshKey] = React.useState(0);
	const { settings } = useTenantSettings();
	const { textGet } = useText();
	// A phone shows one day (moved by arrows, the week strip or a swipe); a
	// desktop shows the week.
	const { isPhone } = useViewport();
	const [pendingOpen, setPendingOpen] = React.useState(false);

	const weekStart = React.useMemo(
		() => startOfWeek(currentDate, { weekStartsOn: 1 }),
		[currentDate],
	);
	const weekEnd = React.useMemo(
		() => endOfWeek(currentDate, { weekStartsOn: 1 }),
		[currentDate],
	);
	const weekDays = eachDayOfInterval({ start: weekStart, end: weekEnd });
	const visibleDays = isPhone ? [currentDate] : weekDays;
	const step = (direction: 1 | -1) =>
		setCurrentDate((date) =>
			isPhone ? addDays(date, direction) : addWeeks(date, direction),
		);
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

	// Open each day (or week) where the work is: today an hour before now,
	// any other day at 08:00, instead of at the top of the grid.
	const scrollRef = React.useRef<HTMLDivElement>(null);
	React.useEffect(() => {
		const hour = isToday(currentDate) ? new Date().getHours() - 1 : 8;
		scrollRef.current?.scrollTo({
			top: Math.max(0, hour - START_HOUR) * HOUR_HEIGHT,
		});
	}, [currentDate]);

	const sensors = useDragSensors();
	// A swipe that ends a drag must not also change the day.
	const dragging = React.useRef(false);
	const swipeStart = React.useRef<{ x: number; y: number } | null>(null);

	function handleTouchStart(event: React.TouchEvent) {
		const touch = event.touches[0];
		swipeStart.current = { x: touch.clientX, y: touch.clientY };
	}

	function handleTouchEnd(event: React.TouchEvent) {
		const start = swipeStart.current;
		swipeStart.current = null;
		if (!isPhone || !start || dragging.current) return;
		const touch = event.changedTouches[0];
		const dx = touch.clientX - start.x;
		const dy = touch.clientY - start.y;
		if (Math.abs(dx) > 60 && Math.abs(dx) > 2 * Math.abs(dy)) {
			step(dx < 0 ? 1 : -1);
		}
	}

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

		const target = computeSnappedTarget(appt, over, delta, visibleDays);
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
		dragging.current = false;
		setDragGhost(null);
	}

	function handleDragEnd(event: DragEndEvent) {
		dragging.current = false;
		setDragGhost(null);
		const { active, over, delta } = event;
		if (!over) return;

		const apptId = Number(active.id);
		const appt = appointments.find((a) => a.ID === apptId);
		if (!appt) return;

		const target = computeSnappedTarget(appt, over, delta, visibleDays);
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
		setSuggestedPatient(null);
		setShowFormDialog(true);
	}

	function handleEditAppointment(appt: Appointment) {
		setEditTarget(appt);
		setCreateDefaults({});
		setSuggestedPatient(null);
		setShowFormDialog(true);
	}

	function handleNewAppointment() {
		setEditTarget(null);
		setCreateDefaults({ date: new Date() });
		setSuggestedPatient(null);
		setShowFormDialog(true);
	}

	function handleScheduleFromSuggestion(patient: Patient, date: Date) {
		setPendingOpen(false);
		setEditTarget(null);
		setCreateDefaults({ date });
		setSuggestedPatient(patient);
		setShowFormDialog(true);
	}

	function handleFormSuccess() {
		fetchAppointments();
		setPendingRefreshKey((k) => k + 1);
	}

	const showPending = settings.clinical.show_next_appointment;
	const pendingPanel = (className?: string) => (
		<PendingFollowUpsPanel
			refreshKey={pendingRefreshKey}
			onSchedule={handleScheduleFromSuggestion}
			className={className}
		/>
	);

	return (
		<div className="flex h-full min-h-0 flex-col">
			{/* ── Header ─────────────────────────────────── */}
			<div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-3 pb-4">
				<div className="flex items-center gap-4">
					<PageHeader title={<Text uuid="appointments.title" />} />
					<div className="flex items-center gap-1">
						<Button
							variant="outline"
							size="icon"
							aria-label={textGet("appointments.nav.previous")}
							onClick={() => step(-1)}
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
							aria-label={textGet("appointments.nav.next")}
							onClick={() => step(1)}
						>
							<ChevronRight className="h-4 w-4" />
						</Button>
					</div>
				</div>
				<div className="flex items-center gap-2 sm:gap-3">
					<p className="font-medium text-muted-foreground capitalize max-sm:text-sm sm:text-lg">
						{isPhone
							? format(currentDate, "EEEE d MMM", { locale: es })
							: `${format(weekStart, "d MMM", { locale: es })} — ${format(weekEnd, "d MMM yyyy", { locale: es })}`}
					</p>
					{isPhone && showPending && (
						<Button
							variant="outline"
							size="icon"
							aria-label={textGet("appointments.pending.title")}
							onClick={() => setPendingOpen(true)}
						>
							<CalendarClock className="h-4 w-4" />
						</Button>
					)}
					<Button onClick={handleNewAppointment}>
						<Plus className="mr-2 h-4 w-4" />
						<Text uuid="appointments.new" />
					</Button>
				</div>
			</div>

			{/* ── Week strip (phone): pick the day, dots mark appointments ── */}
			{isPhone && (
				<div className="grid grid-cols-7 gap-1 pb-3">
					{weekDays.map((day) => {
						const selected = isSameDay(day, currentDate);
						const busy = appointments.some((a) =>
							isSameDay(new Date(a.date), day),
						);
						return (
							<button
								key={day.toISOString()}
								type="button"
								aria-pressed={selected}
								aria-label={format(day, "EEEE d MMMM", { locale: es })}
								onClick={() => setCurrentDate(day)}
								className={cn(
									"flex flex-col items-center gap-1 rounded-xl py-2 outline-none transition-colors focus-visible:ring-3 focus-visible:ring-ring/50 motion-reduce:transition-none",
									selected
										? "bg-primary text-primary-foreground"
										: "hover:bg-muted",
									!selected && isToday(day) && "text-primary",
								)}
							>
								<span className="text-[11px] uppercase opacity-70">
									{format(day, "EEEEE", { locale: es })}
								</span>
								<span className="text-base font-semibold leading-none">
									{format(day, "d")}
								</span>
								<span
									className={cn(
										"size-1 rounded-full",
										busy && (selected ? "bg-primary-foreground" : "bg-primary"),
									)}
								/>
							</button>
						);
					})}
				</div>
			)}

			{/* ── Calendar Grid + Pending Panel ───────────── */}
			<div className="flex min-h-0 flex-1 gap-4">
				<div className="flex min-w-0 flex-1 flex-col overflow-hidden rounded-xl border bg-card">
					{/* Scrollable area with headers inside */}
					<div
						ref={scrollRef}
						className="min-h-0 flex-1 overflow-auto"
						style={{ scrollbarGutter: "stable" }}
						onTouchStart={handleTouchStart}
						onTouchEnd={handleTouchEnd}
					>
						{/* Day Headers (sticky); the week strip does this on a phone */}
						{!isPhone && (
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
						)}

						{/* Time Grid */}
						<DndContext
							sensors={sensors}
							onDragStart={() => {
								dragging.current = true;
							}}
							onDragMove={handleDragMove}
							onDragEnd={handleDragEnd}
							onDragCancel={handleDragCancel}
						>
							<div
								className={cn(
									"grid relative",
									isPhone
										? "grid-cols-[48px_1fr]"
										: "grid-cols-[60px_repeat(7,1fr)]",
								)}
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
								{visibleDays.map((day) => {
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

				{showPending && !isPhone && pendingPanel()}
			</div>

			{showPending && isPhone && (
				<Sheet open={pendingOpen} onOpenChange={setPendingOpen}>
					<SheetContent side="bottom" className="p-0">
						<SheetTitle className="sr-only">
							{textGet("appointments.pending.title")}
						</SheetTitle>
						{pendingPanel("w-full max-h-[70dvh] rounded-none border-0")}
					</SheetContent>
				</Sheet>
			)}

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
				defaultPatient={suggestedPatient}
				onSuccess={handleFormSuccess}
			/>
		</div>
	);
}
