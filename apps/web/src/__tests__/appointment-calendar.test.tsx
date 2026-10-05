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

	it("offers scheduling with MANAGE_APPOINTMENT", async () => {
		granted = ["READ_APPOINTMENT", "MANAGE_APPOINTMENT"];
		renderWeek("es");
		expect(screen.getByText("appointments.new")).toBeInTheDocument();
		await act(async () => {});
	});
});
