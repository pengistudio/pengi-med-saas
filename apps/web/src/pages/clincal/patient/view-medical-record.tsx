import { DashboardLayout } from "@/sections/template/dashboard-template";
import ViewMedicalRecord from "@/sections/views/clinical/view-medical-record";

const ViewMedicalRecordPage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4">
				<ViewMedicalRecord />
			</main>
		</DashboardLayout>
	);
};

export default ViewMedicalRecordPage;
