import {
	DndContext,
	type DragEndEvent,
	DragOverlay,
	PointerSensor,
	useDraggable,
	useDroppable,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import { useText } from "@pengi/shared";
import {
	Button,
	Popover,
	PopoverContent,
	PopoverTrigger,
	Spinner,
	Text,
} from "@pengi/ui";
import {
	ArrowLeft,
	ArrowRight,
	Check,
	Copy,
	Link,
	Monitor,
	RefreshCw,
} from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	type Appointment,
	generateDisplayToken,
	getTodayAppointments,
	updateAppointmentStatus,
} from "@/api/clinical-service";
import { PageHeader } from "@/components/custom/page-header";
import { SegmentedProgress } from "@/components/custom/segmented-progress";
import {
	STATUS_COLORS,
	STATUS_I18N_KEYS,
} from "@/components/features/appointments/appointment-utils";
import { cn, dateParser } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

type WaitingStatus = "scheduled" | "arrived" | "in_consultation" | "completed";

/** The patient's path through the day, in order. */
const FLOW: WaitingStatus[] = [
	"scheduled",
	"arrived",
	"in_consultation",
	"completed",
];

/** A patient still "to check in" this long after their start time is late. */
const LATE_AFTER_MINUTES = 5;

function laneTitleKey(status: WaitingStatus) {
	return status === "scheduled"
		? "waiting_room.column.scheduled"
		: STATUS_I18N_KEYS[status];
}

/** Minutes since the appointment's start time (negative if still ahead). */
function minutesPastStart(startTime: string, now: Date) {
	const [h, m] = startTime.split(":").map(Number);
	const start = new Date(now);
	start.setHours(h, m, 0, 0);
	return Math.floor((now.getTime() - start.getTime()) / 60000);
}

function useNow(intervalMs = 60_000) {
	const [now, setNow] = React.useState(() => new Date());
	React.useEffect(() => {
		const id = setInterval(() => setNow(new Date()), intervalMs);
		return () => clearInterval(id);
	}, [intervalMs]);
	return now;
}

function AppointmentCard({
	appointment,
	now,
	onMove,
	moving,
	lifted = false,
}: {
	appointment: Appointment;
	now: Date;
	onMove?: (id: number, status: WaitingStatus) => void;
	moving?: boolean;
	lifted?: boolean;
}) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const index = FLOW.indexOf(appointment.status as WaitingStatus);
	const prevStatus = index > 0 ? FLOW[index - 1] : undefined;
	const nextStatus = index < FLOW.length - 1 ? FLOW[index + 1] : undefined;
	const isDone = appointment.status === "completed";

	const patientName = appointment.patient
		? `${appointment.patient.first_name} ${appointment.patient.last_name}`
		: textGet("waiting_room.unknown_patient");

	const late =
		appointment.status === "scheduled"
			? minutesPastStart(appointment.start_time, now)
			: 0;

	return (
		<div
			className={cn(
				"space-y-3 rounded-xl border bg-card p-3",
				lifted ? "shadow-xl ring-1 ring-primary/30" : "shadow-xs",
			)}
		>
			<div className="flex gap-3">
				<div className="w-11 shrink-0 tabular-nums">
					<p
						className={cn(
							"text-sm font-semibold",
							isDone && "text-muted-foreground",
						)}
					>
						{appointment.start_time.slice(0, 5)}
					</p>
					<p className="text-xs text-muted-foreground">
						{appointment.end_time.slice(0, 5)}
					</p>
				</div>
				<div className="min-w-0 flex-1">
					<p
						className={cn(
							"truncate text-sm font-medium",
							isDone && "text-muted-foreground",
						)}
					>
						{patientName}
					</p>
					{appointment.title && (
						<p className="truncate text-xs text-muted-foreground">
							{appointment.title}
						</p>
					)}
					{late >= LATE_AFTER_MINUTES && (
						<p className="mt-0.5 text-xs font-medium text-destructive">
							{textGet("waiting_room.card.late").replace(
								"{count}",
								String(late),
							)}
						</p>
					)}
				</div>
			</div>

			{onMove && (
				<div className="flex items-center gap-1">
					{appointment.patient && (
						<Button
							variant="ghost"
							size="sm"
							className="text-muted-foreground"
							onClick={() =>
								navigate(`/clinical/medical-records/${appointment.patient_id}`)
							}
						>
							<Text uuid="waiting_room.card.records" />
						</Button>
					)}
					<div className="ml-auto flex items-center gap-1">
						{prevStatus && (
							<Button
								variant="ghost"
								size="icon-sm"
								className="text-muted-foreground"
								disabled={moving}
								aria-label={textGet("waiting_room.card.back")}
								title={textGet("waiting_room.card.back")}
								onClick={() => onMove(appointment.ID, prevStatus)}
							>
								<ArrowLeft />
							</Button>
						)}
						{nextStatus && (
							<Button
								size="sm"
								disabled={moving}
								onClick={() => onMove(appointment.ID, nextStatus)}
							>
								<Text uuid={`waiting_room.advance.${nextStatus}`} />
								{moving ? <Spinner /> : <ArrowRight />}
							</Button>
						)}
					</div>
				</div>
			)}
		</div>
	);
}

function DraggableCard(props: React.ComponentProps<typeof AppointmentCard>) {
	const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
		id: props.appointment.ID,
		data: { appointment: props.appointment },
	});

	if (isDragging) {
		return (
			<div
				ref={setNodeRef}
				className="min-h-[108px] rounded-xl border-2 border-dashed border-primary/30 bg-primary/5"
			/>
		);
	}

	return (
		<div
			ref={setNodeRef}
			{...listeners}
			{...attributes}
			className="cursor-grab touch-none rounded-xl outline-none focus-visible:ring-2 focus-visible:ring-ring active:cursor-grabbing"
		>
			<AppointmentCard {...props} />
		</div>
	);
}

function Lane({
	status,
	count,
	children,
}: {
	status: WaitingStatus;
	count: number;
	children: React.ReactNode;
}) {
	const { textGet } = useText();
	const { setNodeRef, isOver } = useDroppable({ id: status });
	const title = textGet(laneTitleKey(status));

	return (
		<section
			ref={setNodeRef}
			aria-label={title}
			className={cn(
				"flex flex-col rounded-2xl bg-muted/50 ring-1 ring-transparent ring-inset transition-colors",
				isOver && "bg-primary/5 ring-primary/30",
			)}
		>
			<header className="flex items-center gap-2 px-4 pt-3.5 pb-2">
				<span
					className={cn("size-2 rounded-full", STATUS_COLORS[status].dot)}
				/>
				<h2 className="text-sm font-semibold">{title}</h2>
				<span className="text-sm text-muted-foreground tabular-nums">
					{count}
				</span>
			</header>
			<div className="flex flex-col gap-2 px-2.5 pb-3">
				{count === 0 ? (
					<p className="rounded-xl border border-dashed border-border px-4 py-5 text-center text-xs text-muted-foreground">
						{textGet(`waiting_room.lane.empty.${status}`)}
					</p>
				) : (
					children
				)}
			</div>
		</section>
	);
}

function TvScreenPopover() {
	const { textGet } = useText();
	const [code, setCode] = React.useState<string | null>(null);
	const [generating, setGenerating] = React.useState(false);
	const [copied, setCopied] = React.useState<"code" | "link" | null>(null);

	const flashCopied = (what: "code" | "link") => {
		setCopied(what);
		setTimeout(() => setCopied(null), 2000);
	};

	// Each generated code replaces the previous one and unpairs any TV using it,
	// so the link reuses the code already on screen.
	const generate = async () => {
		setGenerating(true);
		const res = await generateDisplayToken();
		setGenerating(false);
		if (!res.success || !res.data) return null;
		const token = (res.data as { token: string }).token;
		setCode(token);
		return token;
	};

	const copyLink = async () => {
		const token = code ?? (await generate());
		if (!token) return;
		await navigator.clipboard.writeText(
			`${window.location.origin}/display/waiting-room?token=${token}`,
		);
		flashCopied("link");
	};

	return (
		<Popover>
			<PopoverTrigger render={<Button variant="outline" />}>
				<Monitor />
				{textGet("waiting_room.tv.title")}
			</PopoverTrigger>
			<PopoverContent align="end" className="w-72 space-y-3 p-4">
				<p className="text-xs text-muted-foreground">
					{textGet("waiting_room.tv.pair_hint")}
				</p>
				<div className="flex h-10 items-center justify-between gap-2 rounded-lg bg-muted px-3">
					{code ? (
						<span className="text-2xl font-semibold tracking-[0.25em] tabular-nums">
							{code}
						</span>
					) : (
						<Button
							variant="ghost"
							size="sm"
							className="-ml-2"
							onClick={generate}
							disabled={generating}
						>
							{generating && <Spinner />}
							{textGet("waiting_room.generate_tv_code")}
						</Button>
					)}
					{code && (
						<Button
							variant="ghost"
							size="icon-sm"
							aria-label={textGet("waiting_room.tv.copy_code")}
							onClick={async () => {
								await navigator.clipboard.writeText(code);
								flashCopied("code");
							}}
						>
							{copied === "code" ? (
								<Check className="text-primary" />
							) : (
								<Copy />
							)}
						</Button>
					)}
				</div>
				<div className="flex flex-col gap-1">
					<Button
						variant="outline"
						size="sm"
						className="w-full"
						onClick={copyLink}
						disabled={generating}
					>
						{copied === "link" ? <Check className="text-primary" /> : <Link />}
						{textGet(
							copied === "link"
								? "waiting_room.tv.link_copied"
								: "waiting_room.copy_tv_link",
						)}
					</Button>
					{code && (
						<Button
							variant="ghost"
							size="sm"
							className="w-full text-muted-foreground"
							onClick={generate}
							disabled={generating}
						>
							{textGet("waiting_room.tv.regenerate")}
						</Button>
					)}
				</div>
			</PopoverContent>
		</Popover>
	);
}

const WaitingRoomPage = () => {
	const { textGet } = useText();
	const now = useNow();
	const [appointments, setAppointments] = React.useState<Appointment[]>([]);
	const [loading, setLoading] = React.useState(true);
	const [moving, setMoving] = React.useState<number | null>(null);
	const [draggedAppointment, setDraggedAppointment] =
		React.useState<Appointment | null>(null);

	const sensors = useSensors(
		useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
	);

	const load = React.useCallback(() => {
		setLoading(true);
		getTodayAppointments()
			.then((res) => {
				if (res.success && res.data) setAppointments(res.data as Appointment[]);
			})
			.finally(() => setLoading(false));
	}, []);

	React.useEffect(() => {
		load();
	}, [load]);

	const handleMove = async (id: number, status: WaitingStatus) => {
		const previous = appointments;
		setAppointments((prev) =>
			prev.map((a) => (a.ID === id ? { ...a, status } : a)),
		);
		setMoving(id);
		const res = await updateAppointmentStatus(id, status);
		if (res.success && res.data) {
			setAppointments((prev) =>
				prev.map((a) => (a.ID === id ? (res.data as Appointment) : a)),
			);
		} else {
			setAppointments(previous);
		}
		setMoving(null);
	};

	const handleDragEnd = (event: DragEndEvent) => {
		setDraggedAppointment(null);
		const { active, over } = event;
		if (!over) return;
		const appointment = active.data.current?.appointment as Appointment;
		const targetStatus = over.id as WaitingStatus;
		if (appointment && appointment.status !== targetStatus) {
			handleMove(appointment.ID, targetStatus);
		}
	};

	const byStatus = React.useMemo(() => {
		const map: Record<WaitingStatus, Appointment[]> = {
			scheduled: [],
			arrived: [],
			in_consultation: [],
			completed: [],
		};
		const sorted = [...appointments].sort((a, b) =>
			a.start_time.localeCompare(b.start_time),
		);
		for (const a of sorted) {
			if (a.status in map) map[a.status as WaitingStatus].push(a);
		}
		return map;
	}, [appointments]);

	const total = FLOW.reduce((sum, s) => sum + byStatus[s].length, 0);
	const summary =
		total === 0
			? textGet("waiting_room.summary.empty")
			: textGet("waiting_room.summary")
					.replace("{done}", String(byStatus.completed.length))
					.replace("{total}", String(total));

	const today = dateParser(new Date(), { dateStyle: "full" });

	return (
		<DashboardLayout>
			<div className="space-y-5">
				<PageHeader
					title={<Text uuid="waiting_room.title" />}
					description={<span className="capitalize">{today}</span>}
					actions={
						<>
							<TvScreenPopover />
							<Button
								variant="outline"
								size="icon"
								onClick={load}
								disabled={loading}
								aria-label={textGet("waiting_room.refresh")}
								title={textGet("waiting_room.refresh")}
							>
								<RefreshCw className={cn(loading && "animate-spin")} />
							</Button>
						</>
					}
				/>

				<SegmentedProgress
					summary={summary}
					segments={FLOW.map((status) => ({
						key: status,
						count: byStatus[status].length,
						className: STATUS_COLORS[status].dot,
					}))}
				/>

				{loading && appointments.length === 0 ? (
					<div className="flex h-48 items-center justify-center">
						<Spinner className="h-8 w-8" />
					</div>
				) : (
					<DndContext
						sensors={sensors}
						onDragStart={(e) =>
							setDraggedAppointment(e.active.data.current?.appointment ?? null)
						}
						onDragEnd={handleDragEnd}
						onDragCancel={() => setDraggedAppointment(null)}
					>
						<div className="grid grid-cols-1 items-start gap-3 md:grid-cols-2 xl:grid-cols-4">
							{FLOW.map((status) => (
								<Lane
									key={status}
									status={status}
									count={byStatus[status].length}
								>
									{byStatus[status].map((a) => (
										<DraggableCard
											key={a.ID}
											appointment={a}
											now={now}
											onMove={handleMove}
											moving={moving === a.ID}
										/>
									))}
								</Lane>
							))}
						</div>
						<DragOverlay dropAnimation={null}>
							{draggedAppointment && (
								<div className="rotate-1">
									<AppointmentCard
										appointment={draggedAppointment}
										now={now}
										lifted
									/>
								</div>
							)}
						</DragOverlay>
					</DndContext>
				)}
			</div>
		</DashboardLayout>
	);
};

export default WaitingRoomPage;
