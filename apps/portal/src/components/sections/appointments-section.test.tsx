import { toDateOnlyString } from "@pengi/shared";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	cancelAppointment,
	cancelBookingRequest,
	getAppointments,
	getBookingRequests,
	type Me,
	type PortalAppointment,
	type PortalAppointmentRequest,
} from "@/api/portal-service";
import { AppointmentsSection } from "./appointments-section";

vi.mock("@/api/portal-service", () => ({
	getAppointments: vi.fn(),
	getBookingRequests: vi.fn(),
	cancelAppointment: vi.fn(),
	cancelBookingRequest: vi.fn(),
}));

const ok = <T,>(data: T) => ({
	success: true as const,
	code: 200,
	message: "",
	data,
});

const tomorrow = (() => {
	const d = new Date();
	d.setDate(d.getDate() + 1);
	return toDateOnlyString(d);
})();

const appointment = (over: Partial<PortalAppointment>): PortalAppointment => ({
	id: 1,
	date: tomorrow,
	start_time: "09:00",
	end_time: "09:30",
	status: "scheduled",
	doctor_name: "ANA PAZ",
	specialty: "",
	specialty_other: "",
	appointment_type: "Consulta",
	location: "",
	can_cancel: true,
	...over,
});

const request = (
	over: Partial<PortalAppointmentRequest>,
): PortalAppointmentRequest => ({
	id: 7,
	status: "pending",
	date: tomorrow,
	start_time: "10:00",
	end_time: "10:30",
	doctor: { full_name: "ANA PAZ", specialty: "", specialty_other: "" },
	type_name: "Consulta",
	reason: "",
	reject_reason: "",
	expires_at: new Date(Date.now() + 3_600_000).toISOString(),
	...over,
});

const clinic = (booking: boolean): Me["clinic"] => ({
	name: "Clínica Sol",
	slug: "sol",
	has_logo: false,
	sections: {
		appointments: true,
		results: true,
		documents: true,
		invoices: true,
	},
	booking: { enabled: booking, choose_by: "doctor" },
});

const renderSection = (booking = true) =>
	render(
		<MemoryRouter>
			<AppointmentsSection clinic={clinic(booking)} />
		</MemoryRouter>,
	);

describe("AppointmentsSection", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(getAppointments).mockResolvedValue(ok([appointment({})]));
		vi.mocked(getBookingRequests).mockResolvedValue(
			ok([
				request({}),
				request({ id: 8, status: "rejected", reject_reason: "Sin cupo" }),
			]),
		);
	});

	it("offers booking and lists pending and turned-down requests", async () => {
		renderSection();
		expect(
			screen.getByRole("link", { name: "*portal.booking.new*" }),
		).toHaveAttribute("href", "/sol/citas/nueva");
		expect(
			await screen.findByText("*portal.booking.pending_title*"),
		).toBeVisible();
		expect(screen.getByText("*portal.booking.status.rejected*")).toBeVisible();
	});

	it("cancels a pending request after confirming in a dialog", async () => {
		vi.mocked(cancelBookingRequest).mockResolvedValue(
			ok(request({ status: "cancelled" })),
		);
		const user = userEvent.setup();
		renderSection();

		await user.click(
			await screen.findByRole("button", {
				name: "*portal.booking.cancel_request*",
			}),
		);
		const dialog = await screen.findByRole("alertdialog");
		expect(dialog).toHaveTextContent("*portal.booking.cancel_request.body*");
		await user.click(
			within(dialog).getByRole("button", {
				name: "*portal.booking.cancel_request*",
			}),
		);

		expect(cancelBookingRequest).toHaveBeenCalledWith(7);
		await waitFor(() => expect(getBookingRequests).toHaveBeenCalledTimes(2));
	});

	it("says to call the clinic when it is too late to cancel", async () => {
		vi.mocked(getAppointments).mockResolvedValue(
			ok([appointment({ can_cancel: false })]),
		);
		renderSection();
		expect(
			await screen.findByText("*portal.booking.call_to_cancel*"),
		).toBeVisible();
		expect(
			screen.queryByRole("button", {
				name: "*portal.booking.cancel_appointment*",
			}),
		).toBeNull();
		expect(cancelAppointment).not.toHaveBeenCalled();
	});

	it("hides booking when the clinic does not take requests", async () => {
		renderSection(false);
		expect(await screen.findByText("*portal.appointments.next*")).toBeVisible();
		expect(screen.queryByText("*portal.booking.new*")).toBeNull();
		expect(getBookingRequests).not.toHaveBeenCalled();
	});
});
