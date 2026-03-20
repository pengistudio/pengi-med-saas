import { ArrowRight, Clock, Link, RefreshCw, UserCheck, Users } from "lucide-react";
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
import { Spinner } from "@/components/ui/spinner";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";
import { dateParser } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

type WaitingStatus = "scheduled" | "arrived" | "in_consultation" | "completed";

interface Column {
	status: WaitingStatus;
	nextStatus?: WaitingStatus;
	icon: React.ComponentType<{ className?: string }>;
	headerClass: string;
}

const COLUMNS: Column[] = [
	{
		status: "scheduled",
		nextStatus: "arrived",
		icon: Clock,
		headerClass: "border-t-blue-500",
	},
	{
		status: "arrived",
		nextStatus: "in_consultation",
		icon: Users,
		headerClass: "border-t-amber-500",
	},
	{
		status: "in_consultation",
		nextStatus: "completed",
		icon: UserCheck,
		headerClass: "border-t-violet-500",
	},
	{
		status: "completed",
		nextStatus: undefined,
		icon: UserCheck,
		headerClass: "border-t-emerald-500",
	},
];

function AppointmentCard({
	appointment,
	nextStatus,
	onAdvance,
	advancing,
}: {
	appointment: Appointment;
	nextStatus?: WaitingStatus;
	onAdvance: (id: number, status: WaitingStatus) => void;
	advancing: boolean;
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
			className={`rounded-lg border-l-4 ${colors.border} bg-card p-3 shadow-sm space-y-2`}
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

const WaitingRoomPage = () => {
	const { textGet } = useText();
	const [appointments, setAppointments] = React.useState<Appointment[]>([]);
	const [loading, setLoading] = React.useState(true);
	const [advancing, setAdvancing] = React.useState<number | null>(null);
	const [copyingLink, setCopyingLink] = React.useState(false);

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

	const handleAdvance = async (id: number, nextStatus: WaitingStatus) => {
		setAdvancing(id);
		const res = await updateAppointmentStatus(id, nextStatus);
		if (res.success && res.data) {
			setAppointments((prev) =>
				prev.map((a) => (a.ID === id ? (res.data as Appointment) : a)),
			);
		}
		setAdvancing(null);
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
			<main className="p-4 md:p-6 space-y-4">
				<div className="flex items-center justify-between">
					<div>
						<h1 className="text-2xl font-bold">
							<Text uuid="waiting_room.title" />
						</h1>
						<p className="text-muted-foreground text-sm capitalize">{today}</p>
					</div>
					<div className="flex gap-2">
					<Button
						variant="outline"
						size="sm"
						onClick={handleCopyLink}
						disabled={copyingLink}
					>
						<Link className="h-4 w-4 mr-2" />
						<Text uuid="waiting_room.copy_tv_link" />
					</Button>
					<Button variant="outline" size="sm" onClick={load} disabled={loading}>
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
									<CardContent className="px-3 pb-3 space-y-2 min-h-24">
										{items.length === 0 ? (
											<p className="text-xs text-muted-foreground text-center py-4">
												<Text uuid="waiting_room.empty_column" />
											</p>
										) : (
											items.map((a) => (
												<AppointmentCard
													key={a.ID}
													appointment={a}
													nextStatus={col.nextStatus}
													onAdvance={handleAdvance}
													advancing={advancing === a.ID}
												/>
											))
										)}
									</CardContent>
								</Card>
							);
						})}
					</div>
				)}
			</main>
		</DashboardLayout>
	);
};

export default WaitingRoomPage;
