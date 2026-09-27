import { useText } from "@pengi/shared";
import { Button, Card, CardContent, CardHeader, CardTitle } from "@pengi/ui";
import { CalendarPlus, Check, Stethoscope } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import type { UpcomingAppointment } from "@/api/clinical-service";
import {
	STATUS_I18N_KEYS,
	timeToMinutes,
} from "@/components/features/appointments/appointment-utils";
import { cn } from "@/lib/utils";

const nowMinutes = () => {
	const now = new Date();
	return now.getHours() * 60 + now.getMinutes();
};

const formatClock = (minutes: number) =>
	`${String(Math.floor(minutes / 60)).padStart(2, "0")}:${String(minutes % 60).padStart(2, "0")}`;

/** Minutes since midnight, refreshed every minute, for the "now" marker. */
function useNowMinutes() {
	const [minutes, setMinutes] = React.useState(nowMinutes);
	React.useEffect(() => {
		const id = window.setInterval(() => setMinutes(nowMinutes()), 60_000);
		return () => window.clearInterval(id);
	}, []);
	return minutes;
}

const DOT_STYLES: Record<string, string> = {
	completed: "border-muted-foreground/40 bg-muted-foreground/40",
	in_consultation: "border-primary bg-primary",
	arrived: "border-amber-500 bg-amber-500",
	scheduled: "border-primary bg-background",
};

function NowMarker({ minutes }: { minutes: number }) {
	const { textGet } = useText();
	return (
		<li
			className="grid grid-cols-[3.5rem_1rem_1fr] items-center gap-3"
			aria-hidden
		>
			<span className="text-right text-xs font-semibold tabular-nums text-primary">
				{formatClock(minutes)}
			</span>
			<span className="mx-auto h-2 w-2 rounded-full bg-primary" />
			<span className="flex items-center gap-2">
				<span className="h-px flex-1 bg-primary" />
				<span className="text-xs font-medium text-primary">
					{textGet("dashboard.agenda.now")}
				</span>
			</span>
		</li>
	);
}

interface TodayAgendaProps {
	appointments: UpcomingAppointment[];
	canStartConsultation: boolean;
	className?: string;
}

/**
 * Today's appointments on a time rail, with a line at the current time. Rows
 * open the patient's record; the next patient gets the primary action.
 */
export function TodayAgenda({
	appointments,
	canStartConsultation,
	className,
}: TodayAgendaProps) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const now = useNowMinutes();

	const attended = appointments.filter((a) => a.status === "completed").length;
	// The now line goes before the first appointment that hasn't started yet.
	const nowIndex = appointments.findIndex(
		(a) => timeToMinutes(a.start_time) > now,
	);
	const nextId = appointments.find(
		(a) => a.status !== "completed" && timeToMinutes(a.end_time) > now,
	)?.id;

	const summary = textGet(
		appointments.length === 1
			? "dashboard.agenda.summary.one"
			: "dashboard.agenda.summary.other",
	)
		.replace("{count}", String(appointments.length))
		.replace("{attended}", String(attended));

	return (
		<Card className={className}>
			<CardHeader className="flex flex-row items-start justify-between gap-4">
				<div className="space-y-1">
					<CardTitle className="text-base">
						{textGet("dashboard.agenda.title")}
					</CardTitle>
					{appointments.length > 0 && (
						<p className="text-sm text-muted-foreground">{summary}</p>
					)}
				</div>
				<Button
					variant="ghost"
					size="sm"
					onClick={() => navigate("/clinical/appointments")}
				>
					{textGet("dashboard.agenda.open_calendar")}
				</Button>
			</CardHeader>
			<CardContent>
				{appointments.length === 0 ? (
					<div className="flex flex-col items-start gap-3 rounded-lg border border-dashed p-6">
						<p className="text-sm font-medium">
							{textGet("dashboard.agenda.empty")}
						</p>
						<p className="text-sm text-muted-foreground">
							{textGet("dashboard.agenda.empty_hint")}
						</p>
						<Button
							size="sm"
							variant="outline"
							onClick={() => navigate("/clinical/appointments")}
						>
							<CalendarPlus className="mr-2 h-4 w-4" />
							{textGet("dashboard.upcoming.schedule_btn")}
						</Button>
					</div>
				) : (
					<ol className="relative space-y-1">
						{/* The rail runs behind the dots of every row. */}
						<span
							aria-hidden
							className="absolute top-3 bottom-3 left-[calc(3.5rem+0.75rem+0.5rem)] w-px -translate-x-1/2 bg-border"
						/>
						{appointments.map((appt, index) => {
							const done = appt.status === "completed";
							const isNext = appt.id === nextId;
							return (
								<React.Fragment key={appt.id}>
									{index === nowIndex && <NowMarker minutes={now} />}
									<li className="grid grid-cols-[3.5rem_1rem_1fr] items-center gap-3">
										<span
											className={cn(
												"text-right text-sm tabular-nums",
												done ? "text-muted-foreground" : "font-semibold",
											)}
										>
											{appt.start_time}
										</span>
										<span
											className={cn(
												"relative z-10 mx-auto flex h-3 w-3 items-center justify-center rounded-full border-2",
												DOT_STYLES[appt.status] ?? DOT_STYLES.scheduled,
											)}
										/>
										<div
											className={cn(
												"flex min-w-0 items-center gap-3 rounded-lg px-3 py-2",
												isNext && "bg-primary/5 ring-1 ring-primary/20",
											)}
										>
											<button
												type="button"
												onClick={() =>
													navigate(
														`/clinical/medical-records/${appt.patient_id}`,
													)
												}
												className="min-w-0 flex-1 rounded text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
											>
												<p
													className={cn(
														"truncate text-sm font-medium hover:underline",
														done && "text-muted-foreground",
													)}
												>
													{appt.patient_name}
												</p>
												<p className="truncate text-xs text-muted-foreground">
													{appt.title}
												</p>
											</button>
											{done ? (
												<span className="flex shrink-0 items-center gap-1 text-xs text-muted-foreground">
													<Check className="h-3.5 w-3.5" />
													{textGet(STATUS_I18N_KEYS.completed)}
												</span>
											) : (
												<>
													{appt.status !== "scheduled" && (
														<span className="hidden shrink-0 text-xs font-medium text-muted-foreground sm:inline">
															{textGet(
																STATUS_I18N_KEYS[appt.status] ?? appt.status,
															)}
														</span>
													)}
													{canStartConsultation && (
														<Button
															size="sm"
															variant={isNext ? "default" : "outline"}
															className="shrink-0"
															onClick={() =>
																navigate(
																	`/clinical/medical-records/create?patient_id=${appt.patient_id}`,
																)
															}
														>
															<Stethoscope className="mr-1.5 h-3.5 w-3.5" />
															{textGet("dashboard.agenda.start")}
														</Button>
													)}
												</>
											)}
										</div>
									</li>
								</React.Fragment>
							);
						})}
						{nowIndex === -1 && <NowMarker minutes={now} />}
					</ol>
				)}
			</CardContent>
		</Card>
	);
}
