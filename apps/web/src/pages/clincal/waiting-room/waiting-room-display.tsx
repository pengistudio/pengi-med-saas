import {
	Clock,
	RefreshCw,
	Settings,
	Tv,
	UserCheck,
	Users,
	X,
} from "lucide-react";
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
const VIDEO_URL_KEY = "display_video_url";
const VIDEO_VISIBLE_KEY = "display_video_visible";

function extractYouTubeId(url: string): string | null {
	try {
		const u = new URL(url);
		if (u.hostname.includes("youtu.be")) return u.pathname.slice(1);
		if (u.hostname.includes("youtube.com")) {
			if (u.pathname.startsWith("/embed/")) return u.pathname.split("/")[2];
			return u.searchParams.get("v");
		}
	} catch {
		// not a valid URL
	}
	return null;
}

function buildEmbedUrl(videoId: string): string {
	return `https://www.youtube.com/embed/${videoId}?autoplay=1&mute=1&loop=1&playlist=${videoId}&controls=0&modestbranding=1&rel=0`;
}

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

function useWakeLock() {
	const lockRef = React.useRef<WakeLockSentinel | null>(null);

	const acquire = React.useCallback(async () => {
		try {
			if ("wakeLock" in navigator) {
				lockRef.current = await navigator.wakeLock.request("screen");
			}
		} catch {
			// Wake Lock not supported or denied — silently ignore
		}
	}, []);

	React.useEffect(() => {
		acquire();
		const onVisibilityChange = () => {
			if (document.visibilityState === "visible") acquire();
		};
		document.addEventListener("visibilitychange", onVisibilityChange);
		return () => {
			document.removeEventListener("visibilitychange", onVisibilityChange);
			lockRef.current?.release();
		};
	}, [acquire]);
}

function VideoPanel({ embedUrl }: { embedUrl: string }) {
	return (
		<div className="w-[380px] shrink-0 flex flex-col bg-black rounded-xl overflow-hidden">
			<iframe
				src={embedUrl}
				className="w-full flex-1"
				allow="autoplay; encrypted-media"
				allowFullScreen={false}
				title="Display video"
			/>
		</div>
	);
}

function VideoConfigModal({
	currentUrl,
	onSave,
	onClose,
}: {
	currentUrl: string;
	onSave: (url: string) => void;
	onClose: () => void;
}) {
	const [value, setValue] = React.useState(currentUrl);
	const videoId = value ? extractYouTubeId(value) : null;

	return (
		<div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
			<div className="bg-background rounded-2xl shadow-2xl w-full max-w-md p-6 space-y-4">
				<div className="flex items-center justify-between">
					<h2 className="text-lg font-semibold">Configurar video</h2>
					<button
						type="button"
						onClick={onClose}
						className="text-muted-foreground hover:text-foreground"
					>
						<X className="h-5 w-5" />
					</button>
				</div>
				<div className="space-y-2">
					<label className="text-sm text-muted-foreground" htmlFor="video-url">
						URL de YouTube
					</label>
					<input
						id="video-url"
						type="text"
						value={value}
						onChange={(e) => setValue(e.target.value)}
						placeholder="https://www.youtube.com/watch?v=..."
						className="w-full rounded-lg border bg-muted px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
					/>
					{value && !videoId && (
						<p className="text-xs text-destructive">URL de YouTube no válida</p>
					)}
					{videoId && (
						<p className="text-xs text-emerald-600">Video ID: {videoId}</p>
					)}
				</div>
				<div className="flex gap-2 justify-end">
					{currentUrl && (
						<button
							type="button"
							onClick={() => {
								onSave("");
								onClose();
							}}
							className="px-4 py-2 text-sm rounded-lg text-destructive hover:bg-destructive/10"
						>
							Quitar video
						</button>
					)}
					<button
						type="button"
						onClick={onClose}
						className="px-4 py-2 text-sm rounded-lg border hover:bg-muted"
					>
						Cancelar
					</button>
					<button
						type="button"
						disabled={!!value && !videoId}
						onClick={() => {
							onSave(value);
							onClose();
						}}
						className="px-4 py-2 text-sm rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
					>
						Guardar
					</button>
				</div>
			</div>
		</div>
	);
}

const WaitingRoomDisplayPage = () => {
	useWakeLock();
	const { textGet } = useText();
	const [searchParams] = useSearchParams();
	const token = searchParams.get("token") ?? "";
	const [appointments, setAppointments] = React.useState<Appointment[]>([]);
	const [lastUpdated, setLastUpdated] = React.useState<Date | null>(null);
	const [countdown, setCountdown] = React.useState(REFRESH_INTERVAL_MS / 1000);
	const [loading, setLoading] = React.useState(true);
	const [invalidToken, setInvalidToken] = React.useState(false);

	const [videoUrl, setVideoUrl] = React.useState(
		() => localStorage.getItem(VIDEO_URL_KEY) ?? "",
	);
	const [videoVisible, setVideoVisible] = React.useState(
		() => localStorage.getItem(VIDEO_VISIBLE_KEY) !== "false",
	);
	const [showConfig, setShowConfig] = React.useState(false);

	const videoId = videoUrl ? extractYouTubeId(videoUrl) : null;
	const showPanel = videoVisible && !!videoId;

	const handleSaveUrl = (url: string) => {
		setVideoUrl(url);
		localStorage.setItem(VIDEO_URL_KEY, url);
		if (url) setVideoVisible(true);
	};

	const toggleVideo = () => {
		const next = !videoVisible;
		setVideoVisible(next);
		localStorage.setItem(VIDEO_VISIBLE_KEY, String(next));
	};

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
			} else if (res.code === 401 || res.code === 403) {
				setInvalidToken(true);
			}
			setLastUpdated(new Date());
			setLoading(false);
			setCountdown(REFRESH_INTERVAL_MS / 1000);
		});
	}, [token]);

	React.useEffect(() => {
		const onVisible = () => {
			if (document.visibilityState === "visible") load();
		};
		document.addEventListener("visibilitychange", onVisible);
		return () => document.removeEventListener("visibilitychange", onVisible);
	}, [load]);

	React.useEffect(() => {
		load();
	}, [load]);

	React.useEffect(() => {
		const refreshId = setInterval(load, REFRESH_INTERVAL_MS);
		return () => clearInterval(refreshId);
	}, [load]);

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
				<p className="text-2xl text-muted-foreground">
					Enlace inválido o expirado.
				</p>
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
				<div className="flex items-center gap-4">
					<div className="flex items-center gap-2">
						<button
							type="button"
							onClick={() => setShowConfig(true)}
							className="p-2 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
							title="Configurar video"
						>
							<Settings className="h-5 w-5" />
						</button>
						{videoId && (
							<button
								type="button"
								onClick={toggleVideo}
								className="p-2 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
								title={videoVisible ? "Ocultar video" : "Mostrar video"}
							>
								<Tv
									className={`h-5 w-5 ${videoVisible ? "text-primary" : "opacity-50"}`}
								/>
							</button>
						)}
					</div>
					<p className="text-5xl font-mono font-semibold">
						<LiveClock />
					</p>
				</div>
			</header>

			{/* Main */}
			<main className="flex-1 flex gap-6 p-8 min-h-0">
				<div
					className={`flex-1 grid gap-6 ${showPanel ? "grid-cols-3" : "grid-cols-3"}`}
				>
					{DISPLAY_COLUMNS.map((col) => {
						const colors = STATUS_COLORS[col.status];
						const Icon = col.icon;
						const items = byStatus[col.status];

						return (
							<div key={col.status} className="flex flex-col gap-4">
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
								<div className="flex flex-col gap-3 overflow-y-auto">
									{items.length === 0 ? (
										<div className="p-8 text-center text-xl text-muted-foreground/50">
											{textGet("waiting_room.empty_column")}
										</div>
									) : (
										items.map((a) => <PatientCard key={a.ID} appointment={a} />)
									)}
								</div>
							</div>
						);
					})}
				</div>

				{showPanel && videoId && (
					<VideoPanel embedUrl={buildEmbedUrl(videoId)} />
				)}
			</main>

			{/* Footer */}
			<footer className="flex items-center justify-between px-10 py-4 border-t bg-card text-muted-foreground text-base">
				<div className="flex items-center gap-2">
					<RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
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

			{showConfig && (
				<VideoConfigModal
					currentUrl={videoUrl}
					onSave={handleSaveUrl}
					onClose={() => setShowConfig(false)}
				/>
			)}
		</div>
	);
};

export default WaitingRoomDisplayPage;
