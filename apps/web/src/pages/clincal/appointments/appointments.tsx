import AppointmentCalendar from "@/components/features/appointments/appointment-calendar";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const AppointmentsPage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4">
				<AppointmentCalendar />
			</main>
		</DashboardLayout>
	);
};

export default AppointmentsPage;
