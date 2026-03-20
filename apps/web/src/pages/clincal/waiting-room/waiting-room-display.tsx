import { Clock, RefreshCw, UserCheck, Users } from "lucide-react";
import React from "react";
import { useSearchParams } from "react-router";
import {
	type Appointment,
	getTodayAppointmentsPublic,
} from "@/api/clinical-service";
import {
	STATUS_COLORS,
	STATUS_I18N_KEYS,
} from "@/components/features/appointments/appointment-utils";
import { useText } from "@/hooks/use-text";
import { dateParser } from "@/lib/utils";

type DisplayStatus = "scheduled" | "arrived" | "in_consultation";

const DISPLAY_COLUMNS: {
	status: DisplayStatus;
	icon: React.ComponentType<{ className?: string }>;
	borderClass: string;
	bgClass: string;
}[] = [
	{
		status: "scheduled",
		icon: Clock,
		borderClass: "border-blue-500",
		bgClass: "bg-blue-500/10",
	},
	{
		status: "arrived",
		icon: Users,
		borderClass: "border-amber-500",
		bgClass: "bg-amber-500/10",
	},
	{
		status: "in_consultation",
		icon: UserCheck,
		borderClass: "border-violet-500",
		bgClass: "bg-violet-500/10",
	},
];

const REFRESH_INTERVAL_MS = 15_000;

function LiveClock() {
	const [now, setNow] = React.useState(new Date());

	React.useEffect(() => {
		const id = setInterval(() => setNow(new Date()), 1000);
		return () => clearInterval(id);
	}, []);

	const time = now.toLocaleTimeString("es-EC", {
		hour: "2-digit",
		minute: "2-digit",
		second: "2-digit",
		hour12: false,
	});

	return <span className="tabular-nums">{time}</span>;
}

function PatientCard({ appointment }: { appointment: Appointment }) {
	const colors =
		STATUS_COLORS[appointment.status as keyof typeof STATUS_COLORS] ??
		STATUS_COLORS.scheduled;
	const name = appointment.patient
		? `${appointment.patient.first_name} ${appointment.patient.last_name}`
		: "—";

	return (
		<div
			className={`rounded-xl border-l-4 ${colors.border} ${colors.bg} px-5 py-4 space-y-1`}
		>
			<p className="text-2xl font-semibold leading-tight">{name}</p>
			<p className={`text-lg font-medium ${colors.text}`}>
				{appointment.start_time} – {appointment.end_time}
			</p>
			{appointment.title && (
				<p className="text-base text-muted-foreground">{appointment.title}</p>
			)}
		</div>
	);
}

const WaitingRoomDisplayPage = () => {
	const { textGet } = useText();
	const [searchParams] = useSearchParams();
	const token = searchParams.get("token") ?? "";
	const [appointments, setAppointments] = React.useState<Appointment[]>([]);
	const [lastUpdated, setLastUpdated] = React.useState<Date | null>(null);
	const [countdown, setCountdown] = React.useState(REFRESH_INTERVAL_MS / 1000);
	const [loading, setLoading] = React.useState(true);
	const [invalidToken, setInvalidToken] = React.useState(false);

	const load = React.useCallback(() => {
		if (!token) {
			setInvalidToken(true);
			setLoading(false);
			return;
		}
		getTodayAppointmentsPublic(token).then((res) => {
			if (res.success && res.data) {
				setAppointments(res.data as Appointment[]);
				setInvalidToken(false);
			} else {
				setInvalidToken(true);
			}
			setLastUpdated(new Date());
			setLoading(false);
			setCountdown(REFRESH_INTERVAL_MS / 1000);
		});
	}, [token]);

	// Initial load
	React.useEffect(() => {
		load();
	}, [load]);

	// Auto-refresh every 30s
	React.useEffect(() => {
		const refreshId = setInterval(load, REFRESH_INTERVAL_MS);
		return () => clearInterval(refreshId);
	}, [load]);

	// Countdown ticker
	React.useEffect(() => {
		const tickId = setInterval(
			() => setCountdown((c) => (c > 0 ? c - 1 : 0)),
			1000,
		);
		return () => clearInterval(tickId);
	}, []);

	const byStatus = React.useMemo(() => {
		const map: Record<DisplayStatus, Appointment[]> = {
			scheduled: [],
			arrived: [],
			in_consultation: [],
		};
		for (const a of appointments) {
			if (a.status in map) map[a.status as DisplayStatus].push(a);
		}
		return map;
	}, [appointments]);

	const today = dateParser(new Date(), { dateStyle: "full" });

	if (invalidToken) {
		return (
			<div className="min-h-screen bg-background flex items-center justify-center">
				<p className="text-2xl text-muted-foreground">Enlace inválido o expirado.</p>
			</div>
		);
	}

	return (
		<div className="min-h-screen bg-background flex flex-col select-none">
			{/* Header */}
			<header className="flex items-center justify-between px-10 py-6 border-b bg-card shadow-sm">
				<div>
					<h1 className="text-4xl font-bold tracking-tight">
						{textGet("waiting_room.title")}
					</h1>
					<p className="text-xl text-muted-foreground capitalize mt-1">
						{today}
					</p>
				</div>
				<div className="text-right">
					<p className="text-5xl font-mono font-semibold">
						<LiveClock />
					</p>
				</div>
			</header>

			{/* Columns */}
			<main className="flex-1 grid grid-cols-3 gap-6 p-8">
				{DISPLAY_COLUMNS.map((col) => {
					const colors = STATUS_COLORS[col.status];
					const Icon = col.icon;
					const items = byStatus[col.status];

					return (
						<div key={col.status} className="flex flex-col gap-4">
							{/* Column header */}
							<div
								className={`flex items-center gap-3 rounded-xl border-2 ${col.borderClass} ${col.bgClass} px-6 py-4`}
							>
								<Icon className={`h-7 w-7 ${colors.text}`} />
								<span
									className={`text-2xl font-bold uppercase tracking-wide ${colors.text}`}
								>
									{textGet(STATUS_I18N_KEYS[col.status])}
								</span>
								<span
									className={`ml-auto text-3xl font-bold tabular-nums ${colors.text}`}
								>
									{items.length}
								</span>
							</div>

							{/* Cards */}
							<div className="flex flex-col gap-3 overflow-y-auto">
								{items.length === 0 ? (
									<div className="p-8 text-center text-xl text-muted-foreground/50">
										{textGet("waiting_room.empty_column")}
									</div>
								) : (
									items.map((a) => (
										<PatientCard key={a.ID} appointment={a} />
									))
								)}
							</div>
						</div>
					);
				})}
			</main>

			{/* Footer */}
			<footer className="flex items-center justify-between px-10 py-4 border-t bg-card text-muted-foreground text-base">
				<div className="flex items-center gap-2">
					<RefreshCw
						className={`h-4 w-4 ${loading ? "animate-spin" : ""}`}
					/>
					<span>
						{textGet("waiting_room.display.last_updated")}{" "}
						{lastUpdated
							? lastUpdated.toLocaleTimeString("es-EC", {
									hour: "2-digit",
									minute: "2-digit",
									second: "2-digit",
								})
							: "—"}
					</span>
				</div>
				<span>
					{textGet("waiting_room.display.next_refresh")} {countdown}s
				</span>
			</footer>
		</div>
	);
};

export default WaitingRoomDisplayPage;
