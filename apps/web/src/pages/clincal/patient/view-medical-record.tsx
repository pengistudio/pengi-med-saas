import { DashboardLayout } from "@/sections/template/dashboard-template";
import ViewMedicalRecord from "@/sections/views/clinical/view-medical-record";

const ViewMedicalRecordPage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<ViewMedicalRecord />
			</main>
		</DashboardLayout>
	);
};

export default ViewMedicalRecordPage;
