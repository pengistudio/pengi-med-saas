import { useText } from "@pengi/shared";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Form,
	FormInput,
	Spinner,
} from "@pengi/ui";
import React from "react";
import { z } from "zod";
import {
	type Appointment,
	getAppointmentVitalSigns,
	saveAppointmentVitalSigns,
} from "@/api/clinical-service";
import {
	fromVitalSigns,
	toVitalSignsInput,
	VITAL_SIGN_FIELDS,
	type VitalSignsValues,
	vitalSignsSchema,
} from "./vital-signs-schema";

const formSchema = z.object({ vital_signs: vitalSignsSchema });

/**
 * Triage: records an appointment's vital signs before the consultation, without
 * access to the medical record. The doctor finds them filled in when creating
 * the record from this appointment.
 */
export function TriageVitalSignsDialog({
	appointment,
	onClose,
	onSaved,
}: {
	/** The appointment being triaged; null closes the dialog. */
	appointment: Appointment | null;
	onClose: () => void;
	onSaved: (appointmentId: number) => void;
}) {
	const { textGet } = useText();
	// undefined while loading the vital signs already taken.
	const [initial, setInitial] = React.useState<VitalSignsValues | undefined>();
	const [saving, setSaving] = React.useState(false);

	const appointmentId = appointment?.ID;
	React.useEffect(() => {
		setInitial(undefined);
		if (!appointmentId) return;
		let cancelled = false;
		getAppointmentVitalSigns(appointmentId).then((res) => {
			if (cancelled) return;
			setInitial(res.success && res.data ? fromVitalSigns(res.data) : {});
		});
		return () => {
			cancelled = true;
		};
	}, [appointmentId]);

	const patientName = appointment?.patient
		? `${appointment.patient.first_name} ${appointment.patient.last_name}`
		: "";

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!appointment) return;
		setSaving(true);
		const res = await saveAppointmentVitalSigns(
			appointment.ID,
			toVitalSignsInput(values.vital_signs),
		);
		setSaving(false);
		if (res.success) {
			onSaved(appointment.ID);
			onClose();
		}
	}

	return (
		<Dialog open={!!appointment} onOpenChange={(open) => !open && onClose()}>
			<DialogContent className="sm:max-w-lg">
				<DialogHeader>
					<DialogTitle>
						{textGet("form.create_medical_record.vital_signs.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet("waiting_room.vital_signs.description", {
							name: patientName,
						})}
					</DialogDescription>
				</DialogHeader>
				{initial === undefined ? (
					<div className="flex justify-center py-8">
						<Spinner />
					</div>
				) : (
					<Form
						schema={formSchema}
						defaultValues={{ vital_signs: initial }}
						onSubmit={onSubmit}
					>
						{(field) => (
							<>
								<div className="grid grid-cols-2 gap-4">
									{VITAL_SIGN_FIELDS.map((vital) => (
										<FormInput
											key={vital.name}
											field={field}
											name={`vital_signs.${vital.name}`}
											type={vital.name === "blood_pressure" ? "text" : "number"}
											step={"step" in vital ? vital.step : undefined}
											label={textGet(
												`form.create_medical_record.vital_signs.${vital.name}`,
											)}
											placeholder={vital.placeholder}
											endAddon={
												<span className="px-2 text-sm text-muted-foreground">
													{vital.unit}
												</span>
											}
											isOptional
										/>
									))}
								</div>
								<DialogFooter className="mt-6">
									<Button type="button" variant="outline" onClick={onClose}>
										{textGet("common.cancel")}
									</Button>
									<Button type="submit" disabled={saving}>
										{saving && <Spinner />}
										{textGet("common.save")}
									</Button>
								</DialogFooter>
							</>
						)}
					</Form>
				)}
			</DialogContent>
		</Dialog>
	);
}
