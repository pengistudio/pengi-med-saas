import {
	DndContext,
	type DragEndEvent,
	type DragMoveEvent,
} from "@dnd-kit/core";
import { useText } from "@pengi/shared";
import {
	Button,
	DropdownMenu,
	DropdownMenuCheckboxItem,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuTrigger,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
	Sheet,
	SheetContent,
	SheetTitle,
	Text,
	ToggleGroup,
	ToggleGroupItem,
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
import {
	CalendarClock,
	ChevronLeft,
	ChevronRight,
	Plus,
	Users,
} from "lucide-react";
import React from "react";
import { useSearchParams } from "react-router";
import {
	type Appointment,
	getAppointments,
	type Patient,
	updateAppointment,
} from "@/api/clinical-service";
import { PageHeader } from "@/components/custom/page-header";
import {
	computeShading,
	type ShadeRange,
} from "@/components/features/agenda/agenda-utils";
import { useAgendaRange } from "@/components/features/agenda/use-agenda-range";
import { useDragSensors } from "@/hooks/use-drag-sensors";
import usePermission from "@/hooks/use-permission";
import useTenantSettings from "@/hooks/use-tenant-settings";
import { PERMISSIONS } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { findDoctor, useDoctorStatus, useDoctors } from "@/store/doctors-store";
import { AppointmentDetailDialog } from "./appointment-detail-dialog";
import { AppointmentFormDialog } from "./appointment-form-dialog";
import { rememberLanding } from "./appointment-landing";
import {
	END_HOUR,
	HOUR_HEIGHT,
	minutesToTime,
	START_HOUR,
	timeToMinutes,
} from "./appointment-utils";
import { DayColumn, type DragGhost } from "./day-column";
import { PendingFollowUpsPanel } from "./pending-follow-ups-panel";

/**
 * A column of the grid: a day (week view) or a doctor on the shown day (day
 * view). `doctorId` is undefined in the week view, null for "no doctor".
 */
interface CalendarColumn {
	key: string;
	day: Date;
	doctorId?: number | null;
	label?: string;
	color?: string;
	/** Inactive or unknown doctor: shown for its appointments, never assigned new ones. */
	inactive?: boolean;
}

// Given a dragged appointment and the current drop target, computes the
// snapped (15-min) destination column/time — shared by the live ghost preview
// and the final persisted update so both agree on the exact same slot.
function computeSnappedTarget(
	appt: Appointment,
	over: { id: string | number } | null | undefined,
	delta: { x: number; y: number },
	columns: CalendarColumn[],
): { column: CalendarColumn; startTime: string; endTime: string } | null {
	if (!over) return null;
	const column = columns.find((c) => c.key === String(over.id));
	if (!column) return null;
	// An appointment can't be unassigned by dropping it on "no doctor".
	if (column.doctorId === null && appt.doctor_id) return null;
	// Nor reassigned to a doctor who no longer attends (its own column is fine).
	if (column.inactive && column.doctorId !== appt.doctor_id) return null;

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
		column,
		startTime: minutesToTime(newStartMinutes),
		endTime: minutesToTime(newStartMinutes + duration),
	};
}

const MINE = "mine";
const ALL = "all";
/** Key of the "no doctor" column in the day view. */
const NO_DOCTOR = "none";

type View = "week" | "day";
const VIEW_STORAGE_KEY = "agenda-view";

function readStoredView(): View {
	try {
		return localStorage.getItem(VIEW_STORAGE_KEY) === "day" ? "day" : "week";
	} catch {
		return "week";
	}
}

function storeView(view: View) {
	try {
		localStorage.setItem(VIEW_STORAGE_KEY, view);
	} catch {
		// Storage unavailable (private mode): the choice lasts this visit only.
	}
}

const doctorKey = (doctorId: number | null | undefined) =>
	doctorId ? String(doctorId) : NO_DOCTOR;

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
		doctorId?: number | null;
	}>({});
	const [suggestedPatient, setSuggestedPatient] =
		React.useState<Patient | null>(null);
	const [pendingRefreshKey, setPendingRefreshKey] = React.useState(0);
	const { settings } = useTenantSettings();
	const { textGet, formatDate } = useText();
	// A phone shows one day (moved by arrows, the week strip or a swipe); a
	// desktop shows the week.
	const { isPhone } = useViewport();
	const [pendingOpen, setPendingOpen] = React.useState(false);
	// Without MANAGE_APPOINTMENT the agenda is read-only: no create, edit,
	// drag-to-reschedule or status changes.
	const { checkPermission } = usePermission();
	const canManage = checkPermission([
		PERMISSIONS.APPOINTMENTS.PERMISSION_MANAGE_APPOINTMENT,
	]);

	// Doctor filter: "mine" (default for a user with a profile), "all", or a
	// doctor's ID. Only shown when the tenant has more than one active doctor.
	const doctorStatus = useDoctorStatus();
	const { doctors, activeDoctors } = useDoctors();
	const myDoctorId = doctorStatus?.doctor?.ID;
	const [doctorFilter, setDoctorFilter] = React.useState<string | null>(null);
	const showDoctorFilter = activeDoctors.length > 1;
	const effectiveFilter = showDoctorFilter
		? (doctorFilter ?? (myDoctorId ? MINE : ALL))
		: ALL;
	const doctorFilterOptions = [
		...(myDoctorId
			? [{ value: MINE, label: textGet("appointments.filter.mine") }]
			: []),
		{ value: ALL, label: textGet("appointments.filter.all_doctors") },
		...activeDoctors.map((d) => ({ value: String(d.ID), label: d.full_name })),
	];

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

	// "Día" view: one column per doctor on the shown day. Offered once the
	// tenant has doctors; the last choice is remembered on this device.
	const [storedView, setStoredView] = React.useState<View>(readStoredView);
	const canDayView = activeDoctors.length > 0;
	const dayView = canDayView && storedView === "day";
	function changeView(view: View) {
		setStoredView(view);
		storeView(view);
	}
	// Which way the visible range last moved, so new days slide in from that side.
	const [shownDate, setShownDate] = React.useState(currentDate);
	const [slideFrom, setSlideFrom] = React.useState<"left" | "right" | null>(
		null,
	);
	if (!isSameDay(currentDate, shownDate)) {
		setSlideFrom(currentDate > shownDate ? "right" : "left");
		setShownDate(currentDate);
	}
	const slideIn =
		slideFrom &&
		cn(
			"animate-in fade-in-0 duration-(--motion-base) ease-out-soft",
			slideFrom === "right" ? "slide-in-from-right-4" : "slide-in-from-left-4",
		);
	const selectedIndex = weekDays.findIndex((d) => isSameDay(d, currentDate));
	const step = (direction: 1 | -1) =>
		setCurrentDate((date) =>
			isPhone || dayView ? addDays(date, direction) : addWeeks(date, direction),
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
		if (scrollRef.current) {
			scrollRef.current.scrollTop =
				Math.max(0, hour - START_HOUR) * HOUR_HEIGHT;
		}
	}, [currentDate]);

	const sensors = useDragSensors();
	// A swipe that ends a drag must not also change the day.
	const dragging = React.useRef(false);
	const swipeStart = React.useRef<{
		x: number;
		y: number;
		axis?: "x" | "y";
	} | null>(null);
	// The day follows the finger through --swipe-x, written straight to the
	// grid so a swipe doesn't re-render the calendar on every touchmove.
	const gridRef = React.useRef<HTMLDivElement>(null);

	function setSwipeOffset(dx: number | null) {
		const grid = gridRef.current;
		if (!grid) return;
		if (dx === null) {
			delete grid.dataset.swiping;
			grid.style.setProperty("--swipe-x", "0px");
		} else {
			grid.dataset.swiping = "";
			grid.style.setProperty("--swipe-x", `${dx * 0.6}px`);
		}
	}

	function handleTouchStart(event: React.TouchEvent) {
		const touch = event.touches[0];
		swipeStart.current = { x: touch.clientX, y: touch.clientY };
	}

	function handleTouchMove(event: React.TouchEvent) {
		const start = swipeStart.current;
		if (!isPhone || !start || dragging.current) return;
		const touch = event.touches[0];
		const dx = touch.clientX - start.x;
		const dy = touch.clientY - start.y;
		// Lock the axis once the finger has clearly moved, so scrolling the
		// hours never drags the day sideways.
		if (!start.axis) {
			if (Math.abs(dx) < 8 && Math.abs(dy) < 8) return;
			start.axis = Math.abs(dx) > Math.abs(dy) ? "x" : "y";
		}
		if (start.axis === "x") setSwipeOffset(dx);
	}

	function handleTouchCancel() {
		swipeStart.current = null;
		setSwipeOffset(null);
	}

	function handleTouchEnd(event: React.TouchEvent) {
		const start = swipeStart.current;
		swipeStart.current = null;
		setSwipeOffset(null);
		if (!isPhone || !start || dragging.current) return;
		const touch = event.changedTouches[0];
		const dx = touch.clientX - start.x;
		const dy = touch.clientY - start.y;
		if (Math.abs(dx) > 60 && Math.abs(dx) > 2 * Math.abs(dy)) {
			step(dx < 0 ? 1 : -1);
		}
	}

	// Unassigned appointments (created before doctors existed) stay visible
	// under "mine": they may well be the user's.
	const visibleAppointments = appointments.filter((a) => {
		if (effectiveFilter === ALL) return true;
		if (effectiveFilter === MINE)
			return !a.doctor_id || a.doctor_id === myDoctorId;
		return a.doctor_id === Number(effectiveFilter);
	});

	function getAppointmentsForDay(day: Date) {
		return visibleAppointments.filter((a) => isSameDay(new Date(a.date), day));
	}

	// Working hours and blocks of the active doctors for the loaded week (one
	// request per week), used to shade the grid.
	const agendaRange = useAgendaRange(
		format(weekStart, "yyyy-MM-dd"),
		format(weekEnd, "yyyy-MM-dd"),
		canDayView,
	);

	/** What to shade in a doctor's column (null: the "no doctor" column). */
	function shadingFor(
		doctorId: number | null,
		day: Date,
	): ShadeRange[] | undefined {
		if (!agendaRange) return undefined;
		const date = format(day, "yyyy-MM-dd");
		if (doctorId === null) {
			const clinic = agendaRange.clinic.find((c) => c.date === date);
			return computeShading({
				ranges: [],
				blocks: clinic?.blocks ?? [],
				hasSchedule: false,
			});
		}
		const agenda = agendaRange.doctors.find((d) => d.doctor_id === doctorId);
		if (!agenda) return undefined;
		const agendaDay = agenda.days.find((d) => d.date === date);
		return computeShading({
			ranges: agendaDay?.ranges ?? [],
			blocks: agendaDay?.blocks ?? [],
			hasSchedule: agenda.has_schedule,
		});
	}

	// The week view shades by one doctor's schedule when the agenda shows a
	// single doctor: the filtered one, or the only active one.
	const weekShadeDoctor =
		effectiveFilter === MINE
			? myDoctorId
			: effectiveFilter !== ALL
				? Number(effectiveFilter)
				: activeDoctors.length === 1
					? activeDoctors[0].ID
					: undefined;

	// Day view columns: active doctors working that day (every active doctor
	// while no one has a schedule), any doctor with appointments that day, and
	// "no doctor" when some appointment has none.
	const shownDayKey = format(currentDate, "yyyy-MM-dd");
	const dayAppointments = appointments.filter((a) =>
		isSameDay(new Date(a.date), currentDate),
	);
	const anySchedule = agendaRange?.doctors.some((d) => d.has_schedule) ?? false;
	const dayColumns: CalendarColumn[] = [
		...doctors
			.filter((d) => {
				if (dayAppointments.some((a) => a.doctor_id === d.ID)) return true;
				if (!d.active) return false;
				if (!anySchedule) return true;
				const agenda = agendaRange?.doctors.find((x) => x.doctor_id === d.ID);
				return !!agenda?.days.find((x) => x.date === shownDayKey)?.ranges
					.length;
			})
			.map((d) => ({
				key: `${shownDayKey}:${d.ID}`,
				day: currentDate,
				doctorId: d.ID,
				label: d.full_name,
				color: d.color,
				inactive: !d.active,
			})),
		// Appointments of a doctor missing from the loaded list keep a column.
		...[
			...new Set(
				dayAppointments
					.map((a) => a.doctor_id)
					.filter((id): id is number => !!id && !findDoctor(doctors, id)),
			),
		].map((id) => ({
			key: `${shownDayKey}:${id}`,
			day: currentDate,
			doctorId: id,
			label:
				dayAppointments.find((a) => a.doctor_id === id)?.doctor?.full_name ||
				textGet("appointments.day.other_doctor"),
			inactive: true,
		})),
		...(dayAppointments.some((a) => !a.doctor_id)
			? [
					{
						key: `${shownDayKey}:${NO_DOCTOR}`,
						day: currentDate,
						doctorId: null,
						label: textGet("appointments.day.no_doctor"),
					},
				]
			: []),
	];
	// Doctors hidden with the selector (desktop), by doctorKey.
	const [hiddenDoctors, setHiddenDoctors] = React.useState<string[]>([]);
	// The one doctor a phone shows in the day view, by doctorKey.
	const [phoneDoctor, setPhoneDoctor] = React.useState<string | null>(null);
	const phoneColumn =
		dayColumns.find((c) => doctorKey(c.doctorId) === phoneDoctor) ??
		dayColumns.find((c) => myDoctorId && c.doctorId === myDoctorId) ??
		dayColumns[0];

	const columns: CalendarColumn[] = dayView
		? isPhone
			? phoneColumn
				? [phoneColumn]
				: []
			: dayColumns.filter((c) => !hiddenDoctors.includes(doctorKey(c.doctorId)))
		: visibleDays.map((day) => ({ key: format(day, "yyyy-MM-dd"), day }));

	// Day view (desktop): one column per doctor, scrolling sideways when many.
	const dayGridStyle: React.CSSProperties = {
		gridTemplateColumns: `60px repeat(${columns.length}, minmax(10rem, 1fr))`,
		minWidth: `calc(60px + ${columns.length} * 10rem)`,
	};

	function columnAppointments(column: CalendarColumn) {
		if (column.doctorId === undefined) return getAppointmentsForDay(column.day);
		return dayAppointments.filter(
			(a) => (a.doctor_id || null) === column.doctorId,
		);
	}

	function columnShading(column: CalendarColumn) {
		if (column.doctorId !== undefined) {
			return shadingFor(column.doctorId, column.day);
		}
		return weekShadeDoctor
			? shadingFor(weekShadeDoctor, column.day)
			: undefined;
	}

	function handleDragMove(event: DragMoveEvent) {
		const { active, over, delta } = event;
		const appt = appointments.find((a) => a.ID === Number(active.id));
		if (!appt) {
			setDragGhost(null);
			return;
		}

		const target = computeSnappedTarget(appt, over, delta, columns);
		if (!target) {
			setDragGhost(null);
			return;
		}

		setDragGhost({
			columnKey: target.column.key,
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

		const target = computeSnappedTarget(appt, over, delta, columns);
		if (!target) return;
		const { startTime: newStartTime, endTime: newEndTime } = target;
		const targetDay = target.column.day;
		const newDate = targetDay.toISOString();
		// Day view: dropping on another doctor's column reassigns it.
		const newDoctorId =
			typeof target.column.doctorId === "number" &&
			target.column.doctorId !== appt.doctor_id
				? target.column.doctorId
				: undefined;

		if (
			isSameDay(new Date(appt.date), targetDay) &&
			appt.start_time === newStartTime &&
			appt.end_time === newEndTime &&
			newDoctorId === undefined
		) {
			return;
		}

		const released = event.active.rect.current.translated;
		if (
			released &&
			(!isSameDay(new Date(appt.date), targetDay) || newDoctorId !== undefined)
		) {
			rememberLanding(apptId, released);
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
							...(newDoctorId !== undefined && {
								doctor_id: newDoctorId,
								doctor: findDoctor(doctors, newDoctorId) ?? null,
							}),
						}
					: a,
			),
		);

		updateAppointment(apptId, {
			date: newDate,
			start_time: newStartTime,
			end_time: newEndTime,
			...(newDoctorId !== undefined && { doctor_id: newDoctorId }),
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

	function handleSlotClick(day: Date, hour: number, doctorId?: number | null) {
		setEditTarget(null);
		setCreateDefaults({
			date: day,
			time: `${hour.toString().padStart(2, "0")}:00`,
			doctorId,
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
			onSchedule={canManage ? handleScheduleFromSuggestion : undefined}
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
						{isPhone || dayView
							? formatDate(currentDate, "weekday-day-month")
							: `${formatDate(weekStart, "day-month")} — ${formatDate(weekEnd, "medium")}`}
					</p>
					{canDayView && (
						<ToggleGroup
							variant="outline"
							value={[dayView ? "day" : "week"]}
							onValueChange={(value) => {
								const next = value[0];
								if (next === "day" || next === "week") changeView(next);
							}}
							aria-label={textGet("appointments.view.label")}
						>
							<ToggleGroupItem value="day">
								<Text uuid="appointments.view.day" />
							</ToggleGroupItem>
							<ToggleGroupItem value="week">
								<Text uuid="appointments.view.week" />
							</ToggleGroupItem>
						</ToggleGroup>
					)}
					{dayView && !isPhone && dayColumns.length > 0 && (
						<DropdownMenu>
							<DropdownMenuTrigger
								render={
									<Button
										variant="outline"
										size="icon"
										aria-label={textGet("appointments.day.doctors")}
										title={textGet("appointments.day.doctors")}
									>
										<Users className="h-4 w-4" />
									</Button>
								}
							/>
							<DropdownMenuContent align="end" className="w-56">
								<DropdownMenuGroup>
									{dayColumns.map((c) => {
										const key = doctorKey(c.doctorId);
										return (
											<DropdownMenuCheckboxItem
												key={key}
												checked={!hiddenDoctors.includes(key)}
												onCheckedChange={(checked) =>
													setHiddenDoctors((hidden) =>
														checked
															? hidden.filter((h) => h !== key)
															: [...hidden, key],
													)
												}
											>
												{c.label}
											</DropdownMenuCheckboxItem>
										);
									})}
								</DropdownMenuGroup>
							</DropdownMenuContent>
						</DropdownMenu>
					)}
					{dayView && isPhone && phoneColumn && dayColumns.length > 1 && (
						<Select
							value={doctorKey(phoneColumn.doctorId)}
							onValueChange={(v) => setPhoneDoctor(v ? String(v) : null)}
						>
							<SelectTrigger
								className="w-auto min-w-36"
								aria-label={textGet("appointments.filter.doctor")}
							>
								<SelectValue>{phoneColumn.label}</SelectValue>
							</SelectTrigger>
							<SelectContent>
								{dayColumns.map((c) => (
									<SelectItem
										key={doctorKey(c.doctorId)}
										value={doctorKey(c.doctorId)}
									>
										{c.label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					)}
					{showDoctorFilter && !dayView && (
						<Select
							value={effectiveFilter}
							onValueChange={(v) => setDoctorFilter(String(v ?? ALL))}
						>
							<SelectTrigger
								className="w-auto min-w-36"
								aria-label={textGet("appointments.filter.doctor")}
							>
								<SelectValue>
									{
										doctorFilterOptions.find((o) => o.value === effectiveFilter)
											?.label
									}
								</SelectValue>
							</SelectTrigger>
							<SelectContent>
								{doctorFilterOptions.map((o) => (
									<SelectItem key={o.value} value={o.value}>
										{o.label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					)}
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
					{canManage && (
						<Button onClick={handleNewAppointment}>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="appointments.new" />
						</Button>
					)}
				</div>
			</div>

			{/* ── Week strip (phone): pick the day, dots mark appointments ── */}
			{isPhone && (
				<div className="relative isolate grid grid-cols-7 gap-1 pb-3">
					{/* Selected-day pill: one element that slides between days */}
					<span
						aria-hidden
						className="absolute top-0 bottom-3 -z-10 w-[calc((100%-1.5rem)/7)] rounded-xl bg-primary transition-[left] duration-(--motion-slow) ease-spring motion-reduce:transition-none"
						style={{
							left: `calc(${selectedIndex} * ((100% - 1.5rem) / 7 + 0.25rem))`,
						}}
					/>
					{weekDays.map((day) => {
						const selected = isSameDay(day, currentDate);
						const busy = visibleAppointments.some((a) =>
							isSameDay(new Date(a.date), day),
						);
						return (
							<button
								key={day.toISOString()}
								type="button"
								aria-pressed={selected}
								aria-label={formatDate(day, "full")}
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
									{formatDate(day, "weekday-narrow")}
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
						onTouchMove={handleTouchMove}
						onTouchEnd={handleTouchEnd}
						onTouchCancel={handleTouchCancel}
					>
						{/* Doctor headers (day view, sticky) */}
						{dayView && !isPhone && columns.length > 0 && (
							<div
								className="sticky top-0 z-20 grid border-b bg-card"
								style={dayGridStyle}
							>
								<div className="border-r" />
								{columns.map((c) => (
									<div
										key={c.key}
										className={cn(
											"flex items-center justify-center gap-2 border-r px-2 py-3 last:border-r-0",
											slideIn,
										)}
									>
										{c.color && (
											<span
												aria-hidden
												className="size-2.5 shrink-0 rounded-full"
												style={{ backgroundColor: c.color }}
											/>
										)}
										<span className="truncate text-sm font-semibold">
											{c.label}
										</span>
									</div>
								))}
							</div>
						)}

						{/* Day Headers (sticky); the week strip does this on a phone */}
						{!isPhone && !dayView && (
							<div className="grid grid-cols-[60px_repeat(7,1fr)] border-b bg-card sticky top-0 z-20 overflow-x-clip">
								<div className="border-r" />
								{weekDays.map((day) => (
									<div
										key={day.toISOString()}
										className={cn(
											"text-center py-3 border-r last:border-r-0",
											isToday(day) && "bg-primary/5",
											slideIn,
										)}
									>
										<p className="text-xs font-medium text-muted-foreground uppercase">
											{formatDate(day, "weekday")}
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

						{dayView && columns.length === 0 && (
							<p className="px-4 py-16 text-center text-sm text-muted-foreground">
								{textGet("appointments.day.empty")}
							</p>
						)}

						{/* Time Grid */}
						{(!dayView || columns.length > 0) && (
							<DndContext
								sensors={sensors}
								onDragStart={() => {
									dragging.current = true;
									setSwipeOffset(null);
								}}
								onDragMove={handleDragMove}
								onDragEnd={handleDragEnd}
								onDragCancel={handleDragCancel}
							>
								<div
									ref={gridRef}
									className={cn(
										"grid relative overflow-x-clip [--swipe-x:0px]",
										isPhone
											? "grid-cols-[48px_1fr]"
											: !dayView && "grid-cols-[60px_repeat(7,1fr)]",
									)}
									style={{
										height: `${hours.length * HOUR_HEIGHT}px`,
										...(dayView && !isPhone && dayGridStyle),
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

									{/* Day (or doctor) Columns */}
									{columns.map((column) => {
										return (
											<DayColumn
												key={column.key}
												day={column.day}
												columnKey={column.key}
												hours={hours}
												appointments={columnAppointments(column)}
												shading={columnShading(column)}
												ghost={
													dragGhost?.columnKey === column.key ? dragGhost : null
												}
												className={cn(
													slideIn,
													isPhone &&
														"translate-x-(--swipe-x) transition-[translate] duration-(--motion-base) ease-spring in-data-swiping:transition-none",
												)}
												onSlotClick={
													canManage
														? (day, hour) =>
																handleSlotClick(day, hour, column.doctorId)
														: undefined
												}
												canReschedule={canManage}
												onAppointmentClick={(appt) => {
													setSelectedAppointment(appt);
													setShowDetailDialog(true);
												}}
											/>
										);
									})}
								</div>
							</DndContext>
						)}
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
				canManage={canManage}
			/>

			<AppointmentFormDialog
				open={canManage && showFormDialog}
				onOpenChange={setShowFormDialog}
				appointment={editTarget}
				defaultDate={createDefaults.date}
				defaultTime={createDefaults.time}
				defaultPatient={suggestedPatient}
				defaultDoctorId={createDefaults.doctorId}
				onSuccess={handleFormSuccess}
			/>
		</div>
	);
}
