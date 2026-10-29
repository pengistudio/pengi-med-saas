import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	type BookingOptions,
	createBookingRequest,
	getBookingSlots,
	getWaitlist,
	joinWaitlist,
	leaveWaitlist,
	type PortalAppointmentRequest,
	type WaitlistEntry,
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
	getWaitlist: vi.fn(),
	joinWaitlist: vi.fn(),
	leaveWaitlist: vi.fn(),
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

const entry: WaitlistEntry = {
	id: 3,
	status: "waiting",
	doctor_id: 10,
	doctor_name: "ANA PAZ",
	type_id: 1,
	type_name: "Consulta",
	created_at: "2026-10-27T10:00:00-05:00",
	expires_at: "2026-11-26T10:00:00-05:00",
};
const noSlots = ok({ days: [] });

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

describe("BookingSteps waitlist", () => {
	beforeEach(() => {
		vi.mocked(getBookingSlots).mockReset();
		vi.mocked(getWaitlist).mockReset();
		vi.mocked(joinWaitlist).mockReset();
		vi.mocked(leaveWaitlist).mockReset();
	});

	it("offers to join the waitlist when there are no times, and joins", async () => {
		vi.mocked(getBookingSlots).mockResolvedValue(noSlots);
		vi.mocked(getWaitlist).mockResolvedValue(ok<WaitlistEntry[]>([]));
		vi.mocked(joinWaitlist).mockResolvedValue(ok(entry));
		const user = userEvent.setup();
		renderFlow();

		await user.click(
			await screen.findByRole("button", { name: "*portal.waitlist.join*" }),
		);
		expect(joinWaitlist).toHaveBeenCalledWith(1, 10);
		expect(await screen.findByText("*portal.waitlist.joined*")).toBeVisible();
		expect(
			screen.getByRole("button", { name: "*portal.waitlist.leave*" }),
		).toBeVisible();
	});

	it("shows the patient already waiting and lets them leave", async () => {
		vi.mocked(getBookingSlots).mockResolvedValue(noSlots);
		vi.mocked(getWaitlist).mockResolvedValue(ok([entry]));
		vi.mocked(leaveWaitlist).mockResolvedValue(
			ok({ ...entry, status: "removed" as const }),
		);
		const user = userEvent.setup();
		renderFlow();

		await user.click(
			await screen.findByRole("button", { name: "*portal.waitlist.leave*" }),
		);
		expect(leaveWaitlist).toHaveBeenCalledWith(3);
		expect(
			await screen.findByRole("button", { name: "*portal.waitlist.join*" }),
		).toBeVisible();
	});

	it("does not offer the waitlist while there are times", async () => {
		vi.mocked(getBookingSlots).mockResolvedValue(slots(["09:00"]));
		renderFlow();

		expect(await screen.findByRole("button", { name: "09:00" })).toBeVisible();
		expect(
			screen.queryByRole("button", { name: "*portal.waitlist.join*" }),
		).toBeNull();
		expect(getWaitlist).not.toHaveBeenCalled();
	});
});
