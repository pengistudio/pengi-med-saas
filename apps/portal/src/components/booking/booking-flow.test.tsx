import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	type BookingOptions,
	createBookingRequest,
	getBookingSlots,
	type PortalAppointmentRequest,
} from "@/api/portal-service";
import { BookingSteps } from "./booking-flow";

vi.mock("@/api/portal-service", () => ({
	BOOKING_ERROR: {
		slotTaken: "E-PORTAL-010",
		tooManyPending: "E-PORTAL-011",
		tooLateToCancel: "E-PORTAL-012",
	},
	getBookingOptions: vi.fn(),
	getBookingSlots: vi.fn(),
	createBookingRequest: vi.fn(),
}));

const consult = { id: 1, name: "Consulta", duration_minutes: 30 };
const doctor = {
	id: 10,
	full_name: "ANA PAZ",
	specialty: "pediatrics",
	specialty_other: "",
	type_ids: [1],
};
const options: BookingOptions = {
	choose_by: "doctor",
	min_date: "2026-10-27",
	max_date: "2026-12-25",
	doctors: [doctor],
	specialties: [
		{ specialty: "pediatrics", specialty_other: "", type_ids: [1] },
	],
	types: [consult],
};

const ok = <T,>(data: T) => ({
	success: true as const,
	code: 200,
	message: "",
	data,
});
const slots = (times: string[]) =>
	ok({ days: [{ date: "2026-10-28", times }] });
const request = (status: PortalAppointmentRequest["status"]) =>
	ok<PortalAppointmentRequest>({
		id: 5,
		status,
		date: "2026-10-28",
		start_time: "09:00",
		end_time: "09:30",
		doctor: {
			full_name: "ANA PAZ",
			specialty: "pediatrics",
			specialty_other: "",
		},
		type_name: "Consulta",
		reason: "",
		reject_reason: "",
		expires_at: "2026-10-28T09:00:00-05:00",
	});

const renderFlow = (o: BookingOptions = options) =>
	render(
		<MemoryRouter>
			<BookingSteps options={o} home="/sol" />
		</MemoryRouter>,
	);

describe("BookingSteps", () => {
	beforeEach(() => {
		vi.mocked(getBookingSlots).mockReset();
		vi.mocked(createBookingRequest).mockReset();
	});

	it("starts on the times when the doctor and type are the only ones", async () => {
		vi.mocked(getBookingSlots).mockResolvedValue(slots(["09:00"]));
		renderFlow();

		expect(
			screen.getByRole("heading", { name: "*portal.booking.step.when*" }),
		).toBeInTheDocument();
		expect(await screen.findByRole("button", { name: "09:00" })).toBeVisible();
		expect(getBookingSlots).toHaveBeenCalledWith(
			1,
			{ doctor_id: 10 },
			"2026-10-27",
			"2026-11-09",
		);
	});

	it("asks for the doctor first when there are several", () => {
		renderFlow({
			...options,
			doctors: [doctor, { ...doctor, id: 11, full_name: "LUIS MORA" }],
		});
		expect(
			screen.getByRole("heading", {
				name: "*portal.booking.step.provider.doctor*",
			}),
		).toBeInTheDocument();
		expect(screen.getByRole("button", { name: /Luis Mora/ })).toBeVisible();
		expect(getBookingSlots).not.toHaveBeenCalled();
	});

	it("on a taken slot says so and loads the times again", async () => {
		vi.mocked(getBookingSlots)
			.mockResolvedValueOnce(slots(["09:00", "09:30"]))
			.mockResolvedValueOnce(slots(["09:30"]));
		vi.mocked(createBookingRequest).mockResolvedValue({
			success: false,
			code: 409,
			message: "taken",
			data: { error_code: "E-PORTAL-010", error_message: "taken" },
		});
		const user = userEvent.setup();
		renderFlow();

		await user.click(await screen.findByRole("button", { name: "09:00" }));
		await user.type(
			screen.getByLabelText(/portal\.booking\.reason\*/),
			"Fiebre",
		);
		await user.click(
			screen.getByRole("button", { name: "*portal.booking.submit*" }),
		);

		expect(createBookingRequest).toHaveBeenCalledWith({
			type_id: 1,
			date: "2026-10-28",
			start_time: "09:00",
			doctor_id: 10,
			reason: "Fiebre",
		});
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"*portal.booking.error.slot_taken*",
		);
		await waitFor(() => expect(getBookingSlots).toHaveBeenCalledTimes(2));
		expect(await screen.findByRole("button", { name: "09:30" })).toBeVisible();
		expect(screen.queryByRole("button", { name: "09:00" })).toBeNull();
	});

	it("shows the pending result after a request the clinic must confirm", async () => {
		vi.mocked(getBookingSlots).mockResolvedValue(slots(["09:00"]));
		vi.mocked(createBookingRequest).mockResolvedValue(request("pending"));
		const user = userEvent.setup();
		renderFlow();

		await user.click(await screen.findByRole("button", { name: "09:00" }));
		await user.click(
			screen.getByRole("button", { name: "*portal.booking.submit*" }),
		);

		expect(
			await screen.findByRole("heading", {
				name: "*portal.booking.done.pending.title*",
			}),
		).toBeInTheDocument();
	});
});
