import AppointmentCalendar from "@/components/features/appointments/appointment-calendar";

const AppointmentsPage = () => {
	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<AppointmentCalendar />
		</main>
	);
};

export default AppointmentsPage;
