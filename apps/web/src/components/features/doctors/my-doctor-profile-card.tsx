import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@pengi/ui";
import { Stethoscope } from "lucide-react";
import React from "react";
import { createMyDoctor, updateMyDoctor } from "@/api/doctors-service";
import { toProfilePayload } from "@/components/features/doctors/doctor-utils";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import DoctorForm, {
	type DoctorFormValues,
} from "@/sections/forms/doctors/doctor-form";
import { useDoctorStatus, useDoctorStore } from "@/store/doctors-store";

/**
 * "My doctor profile" on the profile page: a linked doctor edits their own
 * profile (not its active state or link); a user who attends patients and has
 * none can create it.
 */
export function MyDoctorProfileCard() {
	const { textGet } = useText();
	const { checkPermission } = usePermission();
	const canCreateOwn =
		checkPermission([PERMISSIONS.DOCTORS.PERMISSION_MANAGE_DOCTORS]) ||
		checkPermission([
			PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
		]);
	const status = useDoctorStatus();
	const refresh = useDoctorStore((s) => s.refresh);
	const [creating, setCreating] = React.useState(false);
	const [saving, setSaving] = React.useState(false);

	if (!status) return null;
	const doctor = status.doctor;
	if (!doctor && !canCreateOwn) return null;

	async function handleSubmit(values: DoctorFormValues) {
		setSaving(true);
		const payload = toProfilePayload(values);
		const res = doctor
			? await updateMyDoctor(payload)
			: await createMyDoctor(payload);
		setSaving(false);
		if (res.success) {
			setCreating(false);
			refresh();
		}
	}

	if (!doctor && !creating) {
		return (
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<Stethoscope className="h-5 w-5" />
						{textGet("doctors.my_profile.title")}
					</CardTitle>
					<CardDescription>
						{textGet("doctors.my_profile.empty")}
					</CardDescription>
				</CardHeader>
				<CardContent>
					<Button onClick={() => setCreating(true)}>
						{textGet("doctors.my_profile.create")}
					</Button>
				</CardContent>
			</Card>
		);
	}

	return (
		<DoctorForm
			key={doctor?.UpdatedAt ?? "new"}
			initialData={doctor}
			loading={saving}
			onSubmit={handleSubmit}
			title={textGet("doctors.my_profile.title")}
			description={textGet("doctors.my_profile.description")}
		/>
	);
}
