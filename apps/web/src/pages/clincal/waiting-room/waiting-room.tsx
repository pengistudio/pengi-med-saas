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
import {
	ArrowLeft,
	ArrowRight,
	Check,
	Clock,
	Copy,
	Link,
	Monitor,
	RefreshCw,
	UserCheck,
	Users,
} from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	type Appointment,
	generateDisplayToken,
	getTodayAppointments,
	updateAppointmentStatus,
} from "@/api/clinical-service";
import {
	STATUS_COLORS,
	STATUS_I18N_KEYS,
} from "@/components/features/appointments/appointment-utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover";
import { Spinner } from "@/components/ui/spinner";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";
import { dateParser } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

type WaitingStatus = "scheduled" | "arrived" | "in_consultation" | "completed";

interface Column {
	status: WaitingStatus;
	nextStatus?: WaitingStatus;
	prevStatus?: WaitingStatus;
	icon: React.ComponentType<{ className?: string }>;
	headerClass: string;
}

const COLUMNS: Column[] = [
	{
		status: "scheduled",
		nextStatus: "arrived",
		prevStatus: undefined,
		icon: Clock,
		headerClass: "border-t-blue-500",
	},
	{
		status: "arrived",
		nextStatus: "in_consultation",
		prevStatus: "scheduled",
		icon: Users,
		headerClass: "border-t-amber-500",
	},
	{
		status: "in_consultation",
		nextStatus: "completed",
		prevStatus: "arrived",
		icon: UserCheck,
		headerClass: "border-t-violet-500",
	},
	{
		status: "completed",
		nextStatus: undefined,
		prevStatus: "in_consultation",
		icon: UserCheck,
		headerClass: "border-t-emerald-500",
	},
];

function AppointmentCard({
	appointment,
	nextStatus,
	prevStatus,
	onAdvance,
	advancing,
	isDragging,
}: {
	appointment: Appointment;
	nextStatus?: WaitingStatus;
	prevStatus?: WaitingStatus;
	onAdvance: (id: number, status: WaitingStatus) => void;
	advancing: boolean;
	isDragging?: boolean;
}) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const colors =
		STATUS_COLORS[appointment.status as keyof typeof STATUS_COLORS] ??
		STATUS_COLORS.scheduled;
	const patientName = appointment.patient
		? `${appointment.patient.first_name} ${appointment.patient.last_name}`
		: textGet("waiting_room.unknown_patient");

	return (
		<div
			className={`rounded-lg border-l-4 ${colors.border} bg-card p-3 shadow-sm space-y-2 ${isDragging ? "opacity-50" : ""}`}
		>
			<div className="flex items-start justify-between gap-2">
				<div className="min-w-0">
					<p className="font-medium text-sm truncate">{patientName}</p>
					<p className="text-xs text-muted-foreground">
						{appointment.start_time} – {appointment.end_time}
					</p>
					{appointment.title && (
						<p className="text-xs text-muted-foreground truncate">
							{appointment.title}
						</p>
					)}
				</div>
				<Badge className={`shrink-0 text-xs ${colors.badge}`}>
					{textGet(STATUS_I18N_KEYS[appointment.status] ?? appointment.status)}
				</Badge>
			</div>
			<div className="flex gap-2">
				{appointment.patient && (
					<Button
						variant="ghost"
						size="sm"
						className="h-7 text-xs px-2"
						onClick={() =>
							navigate(`/clinical/medical-records/${appointment.patient_id}`)
						}
					>
						<Text uuid="waiting_room.card.records" />
					</Button>
				)}
				{prevStatus && (
					<Button
						variant="outline"
						size="sm"
						className="h-7 px-2"
						disabled={advancing}
						onClick={() => onAdvance(appointment.ID, prevStatus)}
					>
						<ArrowLeft className="h-3 w-3" />
					</Button>
				)}
				{nextStatus && (
					<Button
						size="sm"
						className="h-7 text-xs px-2 ml-auto"
						disabled={advancing}
						onClick={() => onAdvance(appointment.ID, nextStatus)}
					>
						{advancing ? (
							<Spinner className="h-3 w-3 mr-1" />
						) : (
							<ArrowRight className="h-3 w-3 mr-1" />
						)}
						<Text uuid={`waiting_room.advance.${nextStatus}`} />
					</Button>
				)}
			</div>
		</div>
	);
}

function DraggableCard({
	appointment,
	nextStatus,
	prevStatus,
	onAdvance,
	advancing,
}: {
	appointment: Appointment;
	nextStatus?: WaitingStatus;
	prevStatus?: WaitingStatus;
	onAdvance: (id: number, status: WaitingStatus) => void;
	advancing: boolean;
}) {
	const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
		id: appointment.ID,
		data: { appointment },
	});

	return (
		<div
			ref={setNodeRef}
			{...listeners}
			{...attributes}
			className="cursor-grab active:cursor-grabbing touch-none"
		>
			<AppointmentCard
				appointment={appointment}
				nextStatus={nextStatus}
				prevStatus={prevStatus}
				onAdvance={onAdvance}
				advancing={advancing}
				isDragging={isDragging}
			/>
		</div>
	);
}

function DroppableColumn({
	status,
	children,
}: {
	status: WaitingStatus;
	children: React.ReactNode;
}) {
	const { setNodeRef, isOver } = useDroppable({ id: status });
	return (
		<div
			ref={setNodeRef}
			className={`min-h-24 space-y-2 rounded-md transition-colors ${isOver ? "bg-muted/50 ring-2 ring-primary/30" : ""}`}
		>
			{children}
		</div>
	);
}

const WaitingRoomPage = () => {
	const { textGet } = useText();
	const [appointments, setAppointments] = React.useState<Appointment[]>([]);
	const [loading, setLoading] = React.useState(true);
	const [advancing, setAdvancing] = React.useState<number | null>(null);
	const [displayCode, setDisplayCode] = React.useState<string | null>(null);
	const [generatingCode, setGeneratingCode] = React.useState(false);
	const [copyingLink, setCopyingLink] = React.useState(false);
	const [codeCopied, setCodeCopied] = React.useState(false);
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

	const handleAdvance = async (id: number, status: WaitingStatus) => {
		const previous = appointments;
		setAppointments((prev) =>
			prev.map((a) => (a.ID === id ? { ...a, status } : a)),
		);
		setAdvancing(id);
		const res = await updateAppointmentStatus(id, status);
		if (res.success && res.data) {
			setAppointments((prev) =>
				prev.map((a) => (a.ID === id ? (res.data as Appointment) : a)),
			);
		} else {
			setAppointments(previous);
		}
		setAdvancing(null);
	};

	const handleDragEnd = (event: DragEndEvent) => {
		setDraggedAppointment(null);
		const { active, over } = event;
		if (!over) return;
		const appointment = active.data.current?.appointment as Appointment;
		const targetStatus = over.id as WaitingStatus;
		if (appointment && appointment.status !== targetStatus) {
			handleAdvance(appointment.ID, targetStatus);
		}
	};

	const byStatus = React.useMemo(() => {
		const map: Record<WaitingStatus, Appointment[]> = {
			scheduled: [],
			arrived: [],
			in_consultation: [],
			completed: [],
		};
		for (const a of appointments) {
			if (a.status in map) map[a.status as WaitingStatus].push(a);
		}
		return map;
	}, [appointments]);

	const handleGenerateCode = async () => {
		setGeneratingCode(true);
		const res = await generateDisplayToken();
		if (res.success && res.data) {
			setDisplayCode((res.data as { token: string }).token);
		}
		setGeneratingCode(false);
	};

	const handleCopyLink = async () => {
		setCopyingLink(true);
		const res = await generateDisplayToken();
		if (res.success && res.data) {
			const token = (res.data as { token: string }).token;
			const url = `${window.location.origin}/display/waiting-room?token=${token}`;
			await navigator.clipboard.writeText(url);
		}
		setCopyingLink(false);
	};

	const today = dateParser(new Date(), { dateStyle: "full" });

	return (
		<DashboardLayout>
			<main className="p-4 md:p-6 pt-2 md:pt-3 space-y-4">
				<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
					<div>
						<h1 className="text-2xl font-bold">
							<Text uuid="waiting_room.title" />
						</h1>
						<p className="text-muted-foreground text-sm capitalize">{today}</p>
					</div>
					<div className="flex flex-wrap items-center gap-2">
						<Button
							variant="outline"
							size="sm"
							onClick={handleCopyLink}
							disabled={copyingLink}
						>
							{copyingLink ? (
								<Spinner className="h-4 w-4 mr-2" />
							) : (
								<Link className="h-4 w-4 mr-2" />
							)}
							<Text uuid="waiting_room.copy_tv_link" />
						</Button>
						<Popover>
							<PopoverTrigger
								className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium border border-input bg-background shadow-xs hover:bg-accent hover:text-accent-foreground h-9 px-4 py-2 disabled:pointer-events-none disabled:opacity-50"
								onClick={!displayCode ? handleGenerateCode : undefined}
								disabled={generatingCode}
							>
								{generatingCode ? (
									<Spinner className="h-4 w-4" />
								) : (
									<Monitor className="h-4 w-4" />
								)}
								<Text uuid="waiting_room.generate_tv_code" />
							</PopoverTrigger>
							{displayCode && (
								<PopoverContent className="w-auto p-4" align="end">
									<p className="text-[11px] text-muted-foreground mb-2">
										Visita <span className="font-medium">/display/pair</span> e
										ingresa:
									</p>
									<div className="flex items-center gap-2">
										<p className="font-mono text-2xl font-bold tracking-[0.25em]">
											{displayCode}
										</p>
										<Button
											variant="ghost"
											size="sm"
											className="h-7 w-7 p-0 relative overflow-hidden"
											onClick={async () => {
												await navigator.clipboard.writeText(displayCode);
												setCodeCopied(true);
												setTimeout(() => setCodeCopied(false), 2000);
											}}
										>
											<Copy
												className={`h-3.5 w-3.5 absolute transition-all duration-200 ${codeCopied ? "scale-0 opacity-0" : "scale-100 opacity-100"}`}
											/>
											<Check
												className={`h-3.5 w-3.5 absolute text-emerald-500 transition-all duration-200 ${codeCopied ? "scale-100 opacity-100" : "scale-0 opacity-0"}`}
											/>
										</Button>
									</div>
									<Button
										variant="ghost"
										size="sm"
										className="w-full mt-2 text-xs text-muted-foreground"
										onClick={handleGenerateCode}
										disabled={generatingCode}
									>
										Regenerar código
									</Button>
								</PopoverContent>
							)}
						</Popover>
						<Button
							variant="outline"
							size="sm"
							onClick={load}
							disabled={loading}
						>
							<RefreshCw
								className={`h-4 w-4 mr-2 ${loading ? "animate-spin" : ""}`}
							/>
							<Text uuid="waiting_room.refresh" />
						</Button>
					</div>
				</div>

				{loading ? (
					<div className="flex justify-center items-center h-48">
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
						<div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
							{COLUMNS.map((col) => {
								const colors = STATUS_COLORS[col.status];
								const Icon = col.icon;
								const items = byStatus[col.status];
								return (
									<Card
										key={col.status}
										className={`border-t-4 ${col.headerClass}`}
									>
										<CardHeader className="pb-2 pt-4 px-4">
											<CardTitle className="text-sm font-semibold flex items-center gap-2">
												<Icon className={`h-4 w-4 ${colors.text}`} />
												<span className={colors.text}>
													{textGet(STATUS_I18N_KEYS[col.status])}
												</span>
												<Badge variant="outline" className="ml-auto text-xs">
													{items.length}
												</Badge>
											</CardTitle>
										</CardHeader>
										<CardContent className="px-3 pb-3">
											<DroppableColumn status={col.status}>
												{items.length === 0 ? (
													<p className="text-xs text-muted-foreground text-center py-4">
														<Text uuid="waiting_room.empty_column" />
													</p>
												) : (
													items.map((a) => (
														<DraggableCard
															key={a.ID}
															appointment={a}
															nextStatus={col.nextStatus}
															prevStatus={col.prevStatus}
															onAdvance={handleAdvance}
															advancing={advancing === a.ID}
														/>
													))
												)}
											</DroppableColumn>
										</CardContent>
									</Card>
								);
							})}
						</div>
						<DragOverlay>
							{draggedAppointment && (
								<div className="rotate-1 opacity-95 shadow-xl">
									<AppointmentCard
										appointment={draggedAppointment}
										onAdvance={() => {}}
										advancing={false}
									/>
								</div>
							)}
						</DragOverlay>
					</DndContext>
				)}
			</main>
		</DashboardLayout>
	);
};

export default WaitingRoomPage;
