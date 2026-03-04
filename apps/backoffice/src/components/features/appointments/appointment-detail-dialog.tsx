import { format } from "date-fns";
import { es } from "date-fns/locale";
import { Ban, Check, Clock, Edit, MapPin, Trash2, User } from "lucide-react";
import React from "react";
import {
	type Appointment,
	deleteAppointment,
	updateAppointmentStatus,
} from "@/api/clinical-service";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { useText } from "@/hooks/use-text";
import useToast from "@/hooks/use-toast";
import { cn } from "@/lib/utils";
import { getStatusColor, STATUS_I18N_KEYS } from "./appointment-utils";

// ─── Props ───────────────────────────────────────────────────────────────────

export interface AppointmentDetailDialogProps {
	appointment: Appointment | null;
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onEdit: (appointment: Appointment) => void;
	onRefresh: () => void;
}

// ─── Component ───────────────────────────────────────────────────────────────

export function AppointmentDetailDialog({
	appointment,
	open,
	onOpenChange,
	onEdit,
	onRefresh,
}: AppointmentDetailDialogProps) {
	const [loading, setLoading] = React.useState(false);
	const { errorToast } = useToast();
	const { textGet } = useText();

	if (!appointment) return null;

	const statusColor = getStatusColor(appointment.status);
	const patientName = appointment.patient
		? appointment.patient.full_name ||
			`${appointment.patient.first_name} ${appointment.patient.last_name}`
		: textGet("appointments.patient");

	async function handleCancel() {
		if (!appointment) return;
		setLoading(true);
		const res = await updateAppointmentStatus(appointment.ID, "cancelled");
		if (!res.success) {
			errorToast(null, res.message);
		} else {
			onOpenChange(false);
			onRefresh();
		}
		setLoading(false);
	}

	async function handleComplete() {
		if (!appointment) return;
		setLoading(true);
		const res = await updateAppointmentStatus(appointment.ID, "completed");
		if (!res.success) {
			errorToast(null, res.message);
		} else {
			onOpenChange(false);
			onRefresh();
		}
		setLoading(false);
	}

	async function handleDelete() {
		if (!appointment) return;
		setLoading(true);
		const res = await deleteAppointment(appointment.ID);
		if (!res.success) {
			errorToast(null, res.message);
		} else {
			onOpenChange(false);
			onRefresh();
		}
		setLoading(false);
	}

	const statusLabel =
		textGet(STATUS_I18N_KEYS[appointment.status]) || appointment.status;

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[425px]">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<div className={cn("w-3 h-3 rounded-full", statusColor.dot)} />
						{appointment.title}
					</DialogTitle>
					<DialogDescription>
						{format(new Date(appointment.date), "EEEE, d MMMM yyyy", {
							locale: es,
						})}
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-3 pt-1">
					<Badge className={cn("font-medium", statusColor.badge)}>
						{statusLabel}
					</Badge>

					<div className="flex items-center gap-3 text-sm">
						<Clock className="h-4 w-4 text-muted-foreground shrink-0" />
						<span>
							{appointment.start_time} — {appointment.end_time}
						</span>
					</div>
					<div className="flex items-center gap-3 text-sm">
						<User className="h-4 w-4 text-muted-foreground shrink-0" />
						<span>{patientName}</span>
					</div>
					{appointment.location && (
						<div className="flex items-center gap-3 text-sm">
							<MapPin className="h-4 w-4 text-muted-foreground shrink-0" />
							<span>{appointment.location}</span>
						</div>
					)}
					{appointment.notes && (
						<div className="rounded-lg bg-muted/50 p-3">
							<p className="text-sm text-muted-foreground whitespace-pre-wrap">
								{appointment.notes}
							</p>
						</div>
					)}
				</div>

				{/* Actions */}
				<DialogFooter className="flex-col sm:flex-row flex-wrap gap-2 pt-2">
					{appointment.status === "scheduled" && (
						<>
							<Button
								variant="outline"
								size="sm"
								onClick={() => {
									onOpenChange(false);
									onEdit(appointment);
								}}
								disabled={loading}
							>
								<Edit className="mr-2 h-4 w-4" />
								{textGet("appointments.action.edit")}
							</Button>
							<Button
								variant="outline"
								size="sm"
								className="text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50"
								onClick={handleComplete}
								disabled={loading}
							>
								<Check className="mr-2 h-4 w-4" />
								{textGet("appointments.action.complete")}
							</Button>
							<Button
								variant="outline"
								size="sm"
								className="text-orange-600 hover:text-orange-700 hover:bg-orange-50"
								onClick={handleCancel}
								disabled={loading}
							>
								<Ban className="mr-2 h-4 w-4" />
								{textGet("appointments.action.cancel")}
							</Button>
							<Button
								variant="destructive"
								size="sm"
								onClick={handleDelete}
								disabled={loading}
							>
								<Trash2 className="mr-2 h-4 w-4" />
								{textGet("appointments.action.delete")}
							</Button>
						</>
					)}
					{appointment.status === "cancelled" && (
						<Button
							variant="destructive"
							size="sm"
							onClick={handleDelete}
							disabled={loading}
						>
							<Trash2 className="mr-2 h-4 w-4" />
							{textGet("appointments.action.delete")}
						</Button>
					)}
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
