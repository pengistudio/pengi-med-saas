import { format } from "date-fns";
import { es } from "date-fns/locale";
import { Edit, Plus, Search, User } from "lucide-react";
import React from "react";
import type { z } from "zod";
import {
	type Appointment,
	createAppointment,
	getAllPatientsWithLastFollowUp,
	type Patient,
	updateAppointment,
} from "@/api/clinical-service";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { FormTextArea } from "@/components/forms/form-textarea";
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
import { Input } from "@/components/ui/input";
import { useText } from "@/hooks/use-text";
import useToast from "@/hooks/use-toast";
import { appointmentSchema } from "./appointment-utils";

// ─── Props ───────────────────────────────────────────────────────────────────

export interface AppointmentFormDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	appointment?: Appointment | null;
	defaultDate?: Date;
	defaultTime?: string;
	onSuccess: () => void;
}

// ─── Component ───────────────────────────────────────────────────────────────

export function AppointmentFormDialog({
	open,
	onOpenChange,
	appointment,
	defaultDate,
	defaultTime,
	onSuccess,
}: AppointmentFormDialogProps) {
	const [loading, setLoading] = React.useState(false);
	const [patients, setPatients] = React.useState<Patient[]>([]);
	const [patientSearch, setPatientSearch] = React.useState("");
	const [selectedPatient, setSelectedPatient] = React.useState<Patient | null>(
		null,
	);
	const [showPatientList, setShowPatientList] = React.useState(false);
	const { errorToast } = useToast();
	const { textGet } = useText();
	const isEdit = !!appointment;

	React.useEffect(() => {
		if (open && !isEdit) {
			getAllPatientsWithLastFollowUp().then((res) => {
				if (res.success && res.data) {
					setPatients(res.data.items);
				}
			});
			setSelectedPatient(null);
			setPatientSearch("");
		}
		if (open && isEdit && appointment?.patient) {
			setSelectedPatient(appointment.patient);
			setPatientSearch(
				appointment.patient.full_name ||
					`${appointment.patient.first_name} ${appointment.patient.last_name}`,
			);
		}
	}, [open, isEdit, appointment]);

	const filteredPatients = patients.filter((p) => {
		const name = (
			p.full_name || `${p.first_name} ${p.last_name}`
		).toLowerCase();
		return (
			name.includes(patientSearch.toLowerCase()) ||
			p.document.includes(patientSearch)
		);
	});

	const formDate = appointment
		? new Date(appointment.date)
		: defaultDate || new Date();

	async function onSubmit(values: z.infer<typeof appointmentSchema>) {
		if (!isEdit && !selectedPatient) {
			errorToast(null, textGet("appointments.form.select_patient"));
			return;
		}
		setLoading(true);

		if (isEdit && appointment) {
			const res = await updateAppointment(appointment.ID, {
				title: values.title,
				start_time: values.start_time,
				end_time: values.end_time,
				location: values.location || "",
				notes: values.notes || "",
			});
			if (res.success) {
				onOpenChange(false);
				onSuccess();
			}
		} else if (selectedPatient) {
			const res = await createAppointment({
				patient_id: selectedPatient.ID,
				title: values.title,
				date: formDate.toISOString(),
				start_time: values.start_time,
				end_time: values.end_time,
				location: values.location || "",
				notes: values.notes || "",
			});
			if (res.success) {
				onOpenChange(false);
				onSuccess();
			}
		}
		setLoading(false);
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[500px]">
				<DialogHeader>
					<DialogTitle>
						{isEdit
							? textGet("appointments.edit")
							: textGet("appointments.new")}
					</DialogTitle>
					<DialogDescription>
						{format(formDate, "EEEE, d MMMM yyyy", { locale: es })}
					</DialogDescription>
				</DialogHeader>

				{/* Patient Search (only for create) */}
				{!isEdit && (
					<div className="relative">
						<span className="text-sm font-medium mb-1.5 block">
							{textGet("appointments.patient")} *
						</span>
						<div className="relative">
							<Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
							<Input
								value={patientSearch}
								onChange={(e) => {
									setPatientSearch(e.target.value);
									setShowPatientList(true);
									setSelectedPatient(null);
								}}
								onFocus={() => setShowPatientList(true)}
								placeholder={textGet("appointments.patient.search")}
								className="pl-9"
							/>
						</div>
						{selectedPatient && (
							<Badge className="mt-2" variant="secondary">
								<User className="h-3 w-3 mr-1" />
								{selectedPatient.full_name ||
									`${selectedPatient.first_name} ${selectedPatient.last_name}`}
								{" — "}
								{selectedPatient.document}
							</Badge>
						)}
						{showPatientList &&
							!selectedPatient &&
							patientSearch.length > 0 && (
								<div className="absolute z-50 w-full mt-1 max-h-48 overflow-auto rounded-md border bg-popover shadow-lg">
									{filteredPatients.length === 0 ? (
										<p className="p-3 text-sm text-muted-foreground">
											{textGet("appointments.patient.not_found")}
										</p>
									) : (
										filteredPatients.slice(0, 8).map((p) => (
											<button
												type="button"
												key={p.ID}
												className="w-full text-left px-3 py-2 hover:bg-accent text-sm flex justify-between items-center"
												onClick={() => {
													setSelectedPatient(p);
													setPatientSearch(
														p.full_name || `${p.first_name} ${p.last_name}`,
													);
													setShowPatientList(false);
												}}
											>
												<span className="font-medium">
													{p.full_name || `${p.first_name} ${p.last_name}`}
												</span>
												<span className="text-muted-foreground text-xs">
													{p.document}
												</span>
											</button>
										))
									)}
								</div>
							)}
					</div>
				)}

				<Form
					schema={appointmentSchema}
					defaultValues={{
						title: appointment?.title || "",
						start_time: appointment?.start_time || defaultTime || "09:00",
						end_time: appointment?.end_time
							? appointment.end_time
							: defaultTime
								? `${(Number.parseInt(defaultTime, 10) + 1).toString().padStart(2, "0")}:00`
								: "10:00",
						location: appointment?.location || "",
						notes: appointment?.notes || "",
					}}
					onSubmit={onSubmit}
				>
					{(field) => (
						<div className="space-y-4">
							<FormInput
								field={field}
								name="title"
								label={`${textGet("appointments.form.title_label")} *`}
								placeholder={textGet("appointments.form.title_placeholder")}
							/>
							<div className="grid grid-cols-2 gap-4">
								<FormInput
									field={field}
									name="start_time"
									label={`${textGet("appointments.form.start_time")} *`}
									type="time"
								/>
								<FormInput
									field={field}
									name="end_time"
									label={`${textGet("appointments.form.end_time")} *`}
									type="time"
								/>
							</div>
							<FormInput
								field={field}
								name="location"
								label={textGet("appointments.form.location")}
								placeholder={textGet("appointments.form.location_placeholder")}
								isOptional
							/>
							<FormTextArea
								field={field}
								name="notes"
								label={textGet("appointments.form.notes")}
								placeholder={textGet("appointments.form.notes_placeholder")}
								isOptional
							/>
							<DialogFooter>
								<Button type="submit" disabled={loading}>
									{isEdit ? (
										<>
											<Edit className="mr-2 h-4 w-4" />
											{textGet("appointments.form.save")}
										</>
									) : (
										<>
											<Plus className="mr-2 h-4 w-4" />
											{textGet("appointments.form.create")}
										</>
									)}
								</Button>
							</DialogFooter>
						</div>
					)}
				</Form>
			</DialogContent>
		</Dialog>
	);
}
