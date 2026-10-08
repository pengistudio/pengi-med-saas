import { Loader2 } from "lucide-react";
import React from "react";
import { useNavigate, useParams } from "react-router";
import {
	type Doctor,
	getDoctorById,
	updateDoctor,
} from "@/api/doctors-service";
import { toProfilePayload } from "@/components/features/doctors/doctor-utils";
import DoctorForm, {
	type DoctorFormValues,
} from "@/sections/forms/doctors/doctor-form";
import { useDoctorStore } from "@/store/doctors-store";

export default function EditDoctorPage() {
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const [loading, setLoading] = React.useState(false);
	const [doctor, setDoctor] = React.useState<Doctor | null>(null);
	const refresh = useDoctorStore((s) => s.refresh);

	React.useEffect(() => {
		if (!id) return;
		getDoctorById(Number(id)).then((res) => {
			if (res.success && res.data) setDoctor(res.data);
			else navigate("/doctors");
		});
	}, [id, navigate]);

	const handleSubmit = async (values: DoctorFormValues) => {
		if (!id) return;
		setLoading(true);
		const res = await updateDoctor(Number(id), toProfilePayload(values));
		setLoading(false);
		if (res.success) {
			await refresh();
			navigate("/doctors");
		}
	};

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			{doctor ? (
				<DoctorForm
					initialData={doctor}
					onSubmit={handleSubmit}
					loading={loading}
				/>
			) : (
				<div className="flex h-[50vh] items-center justify-center">
					<Loader2 className="h-8 w-8 animate-spin text-primary" />
				</div>
			)}
		</main>
	);
}
