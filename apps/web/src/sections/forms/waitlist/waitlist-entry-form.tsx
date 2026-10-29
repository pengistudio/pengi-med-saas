import { toDateOnlyString, useText } from "@pengi/shared";
import {
	Button,
	Form,
	FormInput,
	FormSelect,
	FormTextArea,
	Text,
} from "@pengi/ui";
import { Loader2, Save, X } from "lucide-react";
import React from "react";
import { z } from "zod";
import {
	type Appointment,
	getPatientAppointments,
	type Patient,
} from "@/api/clinical-service";
import type {
	WaitlistEntry,
	WaitlistEntryPayload,
} from "@/api/waitlist-service";
import { waitlistPatientName } from "@/api/waitlist-service";
import { typesForDoctor } from "@/components/features/agenda/agenda-utils";
import { PatientSearch, patientName } from "@/sections/whatsapp/patient-search";
import { useAppointmentTypes } from "@/store/agenda-store";
import { useDoctors } from "@/store/doctors-store";

const ANY = "any";
const NOTES_MAX = 500;

const entrySchema = z
	.object({
		patient_id: z.number().int().min(1, "waitlist.form.error.patient"),
		doctor_id: z.string(),
		appointment_type_id: z.string(),
		date_from: z.string(),
		date_to: z.string(),
		appointment_id: z.string(),
		notes: z.string().trim().max(NOTES_MAX, "waitlist.form.error.notes"),
	})
	.superRefine((values, ctx) => {
		if (
			values.date_from &&
			values.date_to &&
			values.date_from > values.date_to
		) {
			ctx.addIssue({
				code: "custom",
				path: ["date_to"],
				message: "waitlist.form.error.date_order",
			});
		}
	});

type EntryValues = z.infer<typeof entrySchema>;

const idOrNull = (value: string) => {
	const n = Number(value);
	return value !== ANY && Number.isInteger(n) && n > 0 ? n : null;
};

/** The API body from the form ("any" = null). */
export function toWaitlistPayload(values: EntryValues): WaitlistEntryPayload {
	return {
		patient_id: values.patient_id,
		doctor_id: idOrNull(values.doctor_id),
		appointment_type_id: idOrNull(values.appointment_type_id),
		date_from: values.date_from,
		date_to: values.date_to,
		appointment_id: idOrNull(values.appointment_id),
		notes: values.notes.trim(),
	};
}

/** The patient's appointments still ahead (to move one earlier). */
function useUpcomingAppointments(patientId: number) {
	const [appointments, setAppointments] = React.useState<Appointment[]>([]);
	React.useEffect(() => {
		if (!patientId) return;
		let active = true;
		getPatientAppointments(patientId).then((res) => {
			if (!active) return;
			const today = toDateOnlyString(new Date());
			setAppointments(
				(res.success ? (res.data ?? []) : []).filter(
					(a) =>
						(a.status === "scheduled" || a.status === "confirmed") &&
						a.date.slice(0, 10) >= today,
				),
			);
		});
		return () => {
			active = false;
		};
	}, [patientId]);
	return patientId ? appointments : [];
}

/**
 * Adds a patient to the waitlist or edits an entry: who (create only), the
 * doctor (or any), the type (or any), an optional date range, an optional
 * appointment to move earlier, and notes.
 */
export function WaitlistEntryForm({
	entry,
	loading,
	onSubmit,
	onCancel,
}: {
	entry?: WaitlistEntry;
	loading?: boolean;
	onSubmit: (payload: WaitlistEntryPayload) => void;
	onCancel: () => void;
}) {
	const { textGet, formatDate } = useText();
	const { activeDoctors } = useDoctors();
	const { types } = useAppointmentTypes();
	const [patient, setPatient] = React.useState<Pick<
		Patient,
		"ID" | "first_name" | "last_name" | "full_name"
	> | null>(
		entry
			? {
					ID: entry.patient.id,
					first_name: entry.patient.first_name,
					last_name: entry.patient.last_name,
					full_name: waitlistPatientName(entry),
				}
			: null,
	);
	const upcoming = useUpcomingAppointments(patient?.ID ?? 0);

	return (
		<Form
			schema={entrySchema}
			onSubmit={(values) => onSubmit(toWaitlistPayload(values))}
			defaultValues={{
				patient_id: entry?.patient.id ?? 0,
				doctor_id: entry?.doctor ? String(entry.doctor.id) : ANY,
				appointment_type_id: entry?.appointment_type
					? String(entry.appointment_type.id)
					: ANY,
				date_from: entry?.date_from ?? "",
				date_to: entry?.date_to ?? "",
				appointment_id: entry?.appointment ? String(entry.appointment.id) : ANY,
				notes: entry?.notes ?? "",
			}}
		>
			{(field) => {
				const doctorId = idOrNull(field.watch("doctor_id"));
				const typeOptions = typesForDoctor(
					types,
					doctorId,
					entry?.appointment_type?.id,
				);
				const patientError = field.formState.errors.patient_id?.message;
				return (
					<div className="space-y-4">
						<div className="grid gap-2">
							<span className="text-sm font-medium">
								<Text uuid="waitlist.form.patient" />
							</span>
							{patient ? (
								<div className="flex items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm">
									<span className="font-medium">{patientName(patient)}</span>
									{!entry && (
										<Button
											type="button"
											variant="ghost"
											size="icon"
											aria-label={textGet("waitlist.form.change_patient")}
											onClick={() => {
												setPatient(null);
												field.setValue("patient_id", 0);
												field.setValue("appointment_id", ANY);
											}}
										>
											<X className="h-4 w-4" />
										</Button>
									)}
								</div>
							) : (
								<PatientSearch
									placeholderKey="waitlist.form.patient_search"
									onPick={(p) => {
										setPatient(p);
										field.setValue("patient_id", p.ID, {
											shouldValidate: true,
										});
									}}
								/>
							)}
							{patientError && (
								<p className="text-sm text-destructive">
									{textGet("waitlist.form.error.patient")}
								</p>
							)}
						</div>
						<div className="grid gap-4 sm:grid-cols-2">
							<FormSelect
								field={field}
								name="doctor_id"
								label={textGet("waitlist.form.doctor")}
								options={[
									{ value: ANY, label: textGet("waitlist.doctor.any") },
									...activeDoctors.map((d) => ({
										value: String(d.ID),
										label: d.full_name,
									})),
									...(entry?.doctor &&
									!activeDoctors.some((d) => d.ID === entry.doctor?.id)
										? [
												{
													value: String(entry.doctor.id),
													label: entry.doctor.full_name,
												},
											]
										: []),
								]}
							/>
							<FormSelect
								field={field}
								name="appointment_type_id"
								label={textGet("waitlist.form.type")}
								options={[
									{ value: ANY, label: textGet("waitlist.type.any") },
									...typeOptions.map((t) => ({
										value: String(t.ID),
										label: t.name,
									})),
								]}
							/>
						</div>
						<div className="grid gap-4 sm:grid-cols-2">
							<FormInput
								field={field}
								name="date_from"
								type="date"
								label={textGet("waitlist.form.date_from")}
								isOptional
							/>
							<FormInput
								field={field}
								name="date_to"
								type="date"
								label={textGet("waitlist.form.date_to")}
								description={textGet("waitlist.form.date_to.description")}
								isOptional
							/>
						</div>
						{patient && (
							<FormSelect
								field={field}
								name="appointment_id"
								label={textGet("waitlist.form.appointment")}
								description={textGet("waitlist.form.appointment.description")}
								isOptional
								options={[
									{
										value: ANY,
										label: textGet("waitlist.form.appointment.none"),
									},
									...upcoming.map((a) => ({
										value: String(a.ID),
										label: `${formatDate(a.date.slice(0, 10), "weekday-day-month")} ${a.start_time}`,
									})),
									...(entry?.appointment &&
									!upcoming.some((a) => a.ID === entry.appointment?.id)
										? [
												{
													value: String(entry.appointment.id),
													label: entry.appointment.date
														? `${formatDate(entry.appointment.date, "weekday-day-month")} ${entry.appointment.start_time}`
														: entry.appointment.start_time,
												},
											]
										: []),
								]}
							/>
						)}
						<FormTextArea
							field={field}
							name="notes"
							label={textGet("waitlist.form.notes")}
							placeholder={textGet("waitlist.form.notes.placeholder")}
							maxLength={NOTES_MAX}
							isOptional
						/>
						<div className="flex justify-end gap-2">
							<Button type="button" variant="outline" onClick={onCancel}>
								{textGet("common.cancel")}
							</Button>
							<Button type="submit" disabled={loading}>
								{loading ? (
									<Loader2 className="mr-2 h-4 w-4 animate-spin" />
								) : (
									<Save className="mr-2 h-4 w-4" />
								)}
								{textGet("waitlist.form.save")}
							</Button>
						</div>
					</div>
				);
			}}
		</Form>
	);
}
