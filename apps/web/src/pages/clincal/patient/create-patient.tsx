import CreatePatientForm from "@/sections/forms/clinical/patient-create-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

export default function CreatePatientPage() {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4">
				<CreatePatientForm />
			</main>
		</DashboardLayout>
	);
}
