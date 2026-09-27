import EditPatientForm from "@/sections/forms/clinical/patient-update-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const EditPatientPage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4">
				<EditPatientForm />
			</main>
		</DashboardLayout>
	);
};

export default EditPatientPage;
