import { useMessageStore } from "@pengi/shared";
import { UiTextProvider } from "@pengi/ui";
import { act, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/api/clinical-service", () => ({
	getAppointments: vi.fn().mockResolvedValue({ success: true, data: [] }),
	updateAppointment: vi.fn(),
}));
let granted: string[] = [];
vi.mock("@/hooks/use-permission", () => ({
	default: () => ({
		checkPermission: (needed: string[]) =>
			needed.every((p) => granted.includes(p)),
	}),
}));
vi.mock("@/hooks/use-tenant-settings", () => ({
	default: () => ({ settings: { clinical: { show_next_appointment: false } } }),
}));
let doctorsMock: {
	ID: number;
	full_name: string;
	active: boolean;
	color: string;
}[] = [];
vi.mock("@/store/doctors-store", () => ({
	useDoctors: () => ({
		doctors: doctorsMock,
		activeDoctors: doctorsMock.filter((d) => d.active),
		loaded: true,
	}),
	useDoctorStatus: () => null,
	findDoctor: (list: { ID: number }[], id?: number | null) =>
		list.find((d) => d.ID === id),
}));
const getAgendaRange = vi.fn();
vi.mock("@/api/agenda-service", () => ({
	getAgendaRange: (...args: unknown[]) => getAgendaRange(...args),
}));
vi.mock("@/components/features/appointments/pending-follow-ups-panel", () => ({
	PendingFollowUpsPanel: () => null,
}));
vi.mock("@/components/features/appointments/appointment-form-dialog", () => ({
	AppointmentFormDialog: () => null,
}));
vi.mock("@/components/features/appointments/appointment-detail-dialog", () => ({
	AppointmentDetailDialog: () => null,
}));

import AppointmentCalendar from "@/components/features/appointments/appointment-calendar";

function renderWeek(lang: "es" | "en") {
	act(() => {
		useMessageStore.setState({ lang, messages: {} });
	});
	// Monday 28 Sep 2026 → week of 28 Sep – 4 Oct.
	return render(
		<UiTextProvider value={{ textGet: (key) => key }}>
			<MemoryRouter initialEntries={["/clinical/appointments?date=2026-09-28"]}>
				<AppointmentCalendar />
			</MemoryRouter>
		</UiTextProvider>,
	);
}

describe("AppointmentCalendar", () => {
	it("names the days and the week in English when the UI is in English", async () => {
		renderWeek("en");

		for (const day of ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]) {
			expect(screen.getByText(day)).toBeInTheDocument();
		}
		expect(screen.getByText(/Sep 28/)).toHaveTextContent("Oct 4, 2026");
		await act(async () => {});
	});

	it("names the days in Spanish when the UI is in Spanish", async () => {
		renderWeek("es");

		expect(screen.getByText(/^lun/)).toBeInTheDocument();
		expect(screen.queryByText("Mon")).not.toBeInTheDocument();
		await act(async () => {});
	});

	it("hides scheduling without MANAGE_APPOINTMENT", async () => {
		granted = ["READ_APPOINTMENT"];
		renderWeek("es");
		expect(screen.queryByText("appointments.new")).not.toBeInTheDocument();
		await act(async () => {});
	});

	it("day view shows one column per doctor working that day", async () => {
		granted = ["READ_APPOINTMENT"];
		doctorsMock = [
			{ ID: 1, full_name: "Ana Mora", active: true, color: "#2563EB" },
			{ ID: 2, full_name: "Beto Ruiz", active: true, color: "#16A34A" },
		];
		getAgendaRange.mockResolvedValue({
			success: true,
			data: {
				from: "2026-09-28",
				to: "2026-10-04",
				doctors: [
					{
						doctor_id: 1,
						active: true,
						has_schedule: true,
						days: [
							{
								date: "2026-09-28",
								ranges: [{ start_time: "08:00", end_time: "12:00" }],
								blocks: [],
							},
						],
					},
					{ doctor_id: 2, active: true, has_schedule: true, days: [] },
				],
				clinic: [],
			},
		});
		localStorage.setItem("agenda-view", "day");
		try {
			renderWeek("es");
			expect(await screen.findByText("Ana Mora")).toBeInTheDocument();
			expect(screen.queryByText("Beto Ruiz")).not.toBeInTheDocument();
			expect(getAgendaRange).toHaveBeenCalledWith("2026-09-28", "2026-10-04");
		} finally {
			localStorage.removeItem("agenda-view");
			doctorsMock = [];
		}
	});

	it("offers scheduling with MANAGE_APPOINTMENT", async () => {
		granted = ["READ_APPOINTMENT", "MANAGE_APPOINTMENT"];
		renderWeek("es");
		expect(screen.getByText("appointments.new")).toBeInTheDocument();
		await act(async () => {});
	});
});
