import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { UpcomingAppointment } from "@/api/clinical-service";

vi.mock("@pengi/shared", async (importOriginal) => ({
	...(await importOriginal<typeof import("@pengi/shared")>()),
	useText: () => ({ textGet: (key: string) => key }),
}));

const mockNavigate = vi.fn();
vi.mock("react-router", () => ({ useNavigate: () => mockNavigate }));

import { TodayAgenda } from "@/sections/dashboard/today-agenda";

const appointment = (
	overrides: Partial<UpcomingAppointment>,
): UpcomingAppointment => ({
	id: 1,
	title: "Control",
	start_time: "08:00",
	end_time: "08:30",
	patient_name: "Ana Mora",
	patient_id: 10,
	status: "scheduled",
	...overrides,
});

const agenda = [
	appointment({
		id: 1,
		start_time: "08:00",
		end_time: "08:30",
		status: "completed",
		patient_name: "Ana Mora",
		patient_id: 10,
	}),
	appointment({
		id: 2,
		start_time: "11:00",
		end_time: "11:30",
		patient_name: "Luis Paz",
		patient_id: 20,
	}),
	appointment({
		id: 3,
		start_time: "12:00",
		end_time: "12:30",
		patient_name: "Eva Ruiz",
		patient_id: 30,
	}),
];

describe("TodayAgenda", () => {
	beforeEach(() => {
		vi.useFakeTimers({ shouldAdvanceTime: true });
		vi.setSystemTime(new Date(2026, 8, 26, 10, 12));
		mockNavigate.mockClear();
	});
	afterEach(() => vi.useRealTimers());

	it("draws the now line between past and upcoming appointments", () => {
		const { container } = render(
			<TodayAgenda appointments={agenda} canStartConsultation />,
		);

		// The now marker is aria-hidden, so it is looked up in the DOM.
		const items = Array.from(container.querySelectorAll("li")).map(
			(li) => li.textContent,
		);
		const nowIndex = items.findIndex((t) =>
			t?.includes("dashboard.agenda.now"),
		);
		expect(items[nowIndex]).toContain("10:12");
		expect(items[nowIndex - 1]).toContain("Ana Mora");
		expect(items[nowIndex + 1]).toContain("Luis Paz");
	});

	it("starts the consultation of the next patient, not of attended ones", () => {
		render(<TodayAgenda appointments={agenda} canStartConsultation />);

		const buttons = screen.getAllByRole("button", {
			name: /dashboard.agenda.start/,
		});
		expect(buttons).toHaveLength(2);
		fireEvent.click(buttons[0]);
		expect(mockNavigate).toHaveBeenCalledWith(
			// Linked to the appointment so the consultation gets its triage vital signs.
			"/clinical/medical-records/create?patient_id=20&appointment_id=2",
		);

		const attended = screen.getByText("Ana Mora").closest("li") as HTMLElement;
		expect(
			within(attended).queryByRole("button", {
				name: /dashboard.agenda.start/,
			}),
		).toBeNull();
	});

	it("opens the patient's record from the row", () => {
		render(<TodayAgenda appointments={agenda} canStartConsultation={false} />);

		fireEvent.click(screen.getByText("Eva Ruiz"));
		expect(mockNavigate).toHaveBeenCalledWith("/clinical/medical-records/30");
		expect(
			screen.queryByRole("button", { name: /dashboard.agenda.start/ }),
		).toBeNull();
	});
});
