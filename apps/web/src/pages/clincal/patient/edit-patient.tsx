import EditPatientForm from "@/sections/forms/clinical/patient-update-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const EditPatientPage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<EditPatientForm />
			</main>
		</DashboardLayout>
	);
};

export default EditPatientPage;
