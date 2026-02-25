import CreateMedicalRecordForm from "@/sections/forms/clinical/medical-record-create-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const CreateMedicalRecordPage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<CreateMedicalRecordForm />
			</main>
		</DashboardLayout>
	);
};

export default CreateMedicalRecordPage;
