import UpdateMedicalRecordForm from "@/sections/forms/clinical/medical-record-update-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const UpdateMedicalRecordPage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4">
				<UpdateMedicalRecordForm />
			</main>
		</DashboardLayout>
	);
};

export default UpdateMedicalRecordPage;
