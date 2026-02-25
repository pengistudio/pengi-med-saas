import CreatePatientForm from "@/sections/forms/clinical/patient-create-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

export default function CreatePatientPage() {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<CreatePatientForm />
			</main>
		</DashboardLayout>
	);
}
