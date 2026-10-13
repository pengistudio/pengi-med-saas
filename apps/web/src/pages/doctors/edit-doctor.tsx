import { useText } from "@pengi/shared";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@pengi/ui";
import { Loader2 } from "lucide-react";
import React from "react";
import { useNavigate, useParams } from "react-router";
import {
	type Doctor,
	getDoctorById,
	updateDoctor,
} from "@/api/doctors-service";
import { DoctorScheduleSection } from "@/components/features/agenda/doctor-schedule-section";
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
	const { textGet } = useText();

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
				<Tabs defaultValue="profile">
					<TabsList>
						<TabsTrigger value="profile">
							{textGet("doctors.tab.profile")}
						</TabsTrigger>
						<TabsTrigger value="schedule">
							{textGet("doctors.tab.schedule")}
						</TabsTrigger>
					</TabsList>
					{/* Both stay mounted: switching tabs keeps unsaved edits. */}
					<TabsContent value="profile" keepMounted>
						<DoctorForm
							initialData={doctor}
							onSubmit={handleSubmit}
							loading={loading}
						/>
					</TabsContent>
					<TabsContent value="schedule" keepMounted>
						<DoctorScheduleSection scope={doctor.ID} />
					</TabsContent>
				</Tabs>
			) : (
				<div className="flex h-[50vh] items-center justify-center">
					<Loader2 className="h-8 w-8 animate-spin text-primary" />
				</div>
			)}
		</main>
	);
}
