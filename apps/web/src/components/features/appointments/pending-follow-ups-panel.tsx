import { parseDateOnly, useText } from "@pengi/shared";
import { Text } from "@pengi/ui";
import { CalendarClock } from "lucide-react";
import React from "react";
import {
	getAllPatientsWithLastFollowUp,
	type Patient,
} from "@/api/clinical-service";
import { cn } from "@/lib/utils";

export interface PendingFollowUpsPanelProps {
	refreshKey: number;
	className?: string;
	/** Absent when the user can't schedule: the list is then read-only. */
	onSchedule?: (patient: Patient, suggestedDate: Date) => void;
}

function suggestedColorClass(date: Date): string {
	const today = new Date();
	today.setHours(0, 0, 0, 0);
	const day = new Date(date);
	day.setHours(0, 0, 0, 0);
	const diffDays = Math.round((day.getTime() - today.getTime()) / 86400000);

	if (diffDays <= 0) {
		return "bg-red-500/15 text-red-700 dark:text-red-400 border-red-400/40";
	}
	if (diffDays === 1) {
		return "bg-amber-500/15 text-amber-700 dark:text-amber-400 border-amber-400/40";
	}
	return "bg-blue-500/15 text-blue-700 dark:text-blue-400 border-blue-400/40";
}

export function PendingFollowUpsPanel({
	refreshKey,
	onSchedule,
	className,
}: PendingFollowUpsPanelProps) {
	const { textGet, formatDate } = useText();
	const [patients, setPatients] = React.useState<Patient[]>([]);
	const [loading, setLoading] = React.useState(false);

	React.useEffect(() => {
		async function loadPendingFollowUps() {
			setLoading(true);
			const res = await getAllPatientsWithLastFollowUp({
				pending_follow_up: true,
			});
			if (res.success && res.data) {
				setPatients(res.data.items);
			}
			setLoading(false);
		}
		loadPendingFollowUps();
	}, [refreshKey]);

	return (
		<aside
			className={cn(
				"flex min-h-0 w-72 shrink-0 flex-col rounded-xl border bg-card",
				className,
			)}
		>
			<div className="flex items-center gap-2 px-4 py-3 border-b">
				<CalendarClock className="h-4 w-4 text-muted-foreground" />
				<h2 className="text-sm font-semibold">
					<Text uuid="appointments.pending.title" />
				</h2>
			</div>
			<div className="flex-1 overflow-auto p-2 space-y-1.5">
				{!loading && patients.length === 0 && (
					<p className="text-xs text-muted-foreground px-2 py-4 text-center">
						<Text uuid="appointments.pending.empty" />
					</p>
				)}
				{patients.map((patient) => {
					const lastRecord = patient.medical_records?.[0];
					const suggestedDate = parseDateOnly(
						lastRecord?.next_appointment_date,
					);
					if (!suggestedDate) return null;
					const dateLabel = formatDate(suggestedDate, "day-month");
					const name =
						patient.full_name || `${patient.first_name} ${patient.last_name}`;

					return (
						<button
							type="button"
							key={patient.ID}
							disabled={!onSchedule}
							onClick={() => onSchedule?.(patient, suggestedDate)}
							className={cn(
								"w-full text-left rounded-lg border p-2.5",
								onSchedule && "hover:bg-accent transition-colors",
							)}
						>
							<div className="flex items-center justify-between gap-2">
								<span className="text-sm font-medium truncate">{name}</span>
								<span
									className={cn(
										"inline-flex shrink-0 rounded-md border px-1.5 py-0.5 text-[11px] font-medium",
										suggestedColorClass(suggestedDate),
									)}
								>
									{dateLabel}
								</span>
							</div>
							<span className="text-[11px] text-muted-foreground">
								{textGet("clinical.patient.appointment.suggested")}
							</span>
						</button>
					);
				})}
			</div>
		</aside>
	);
}
