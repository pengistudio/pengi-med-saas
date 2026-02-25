import AppointmentCalendar from "@/components/features/appointments/appointment-calendar";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const AppointmentsPage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<AppointmentCalendar />
			</main>
		</DashboardLayout>
	);
};

export default AppointmentsPage;
