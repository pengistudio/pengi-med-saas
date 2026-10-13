import React from "react";
import { useNavigate } from "react-router";
import { createDoctor } from "@/api/doctors-service";
import { toProfilePayload } from "@/components/features/doctors/doctor-utils";
import { useLinkableUsers } from "@/components/features/doctors/use-linkable-users";
import DoctorForm, {
	type DoctorFormValues,
	NO_USER,
} from "@/sections/forms/doctors/doctor-form";
import { useDoctorStore } from "@/store/doctors-store";

export default function CreateDoctorPage() {
	const navigate = useNavigate();
	const [loading, setLoading] = React.useState(false);
	const refresh = useDoctorStore((s) => s.refresh);
	const userOptions = useLinkableUsers();

	const handleSubmit = async (values: DoctorFormValues) => {
		setLoading(true);
		const res = await createDoctor({
			...toProfilePayload(values),
			user_id:
				values.user_id && values.user_id !== NO_USER
					? Number(values.user_id)
					: null,
		});
		setLoading(false);
		if (res.success) {
			await refresh();
			navigate("/doctors");
		}
	};

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<DoctorForm
				onSubmit={handleSubmit}
				loading={loading}
				userOptions={userOptions}
			/>
		</main>
	);
}
