import { AppTextBridge, LanguageProvider } from "@pengi/shared";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	type BookingOptions,
	type Clinic,
	createNewPatientRequest,
	getBookingOptions,
	guestSlotsLoader,
	requestNewPatientCode,
	verifyNewPatientCode,
} from "@/api/portal-service";
import { NewPatientFlow } from "./new-patient-flow";

vi.mock("@/api/portal-service", () => ({
	BOOKING_ERROR: { slotTaken: "E-PORTAL-010", tooManyPending: "E-PORTAL-011" },
	NEW_PATIENT_ERROR: {
		invalidContact: "E-PORTAL-005",
		captcha: "E-PORTAL-008",
		notAccepted: "E-PORTAL-015",
		invalidData: "E-PORTAL-016",
		alreadyPending: "E-PORTAL-017",
		clinicFull: "E-PORTAL-018",
	},
	getBookingOptions: vi.fn(),
	getBookingSlots: vi.fn(),
	guestSlotsLoader: vi.fn(),
	requestNewPatientCode: vi.fn(),
	verifyNewPatientCode: vi.fn(),
	createNewPatientRequest: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
	env: { VITE_API_URL: "http://localhost", VITE_TURNSTILE_SITE_KEY: undefined },
}));

const ok = <T,>(data: T) => ({
	success: true as const,
	code: 200,
	message: "",
	data,
});
const fail = (code: number, errorCode: string) => ({
	success: false as const,
	code,
	message: "server message",
	data: { error_code: errorCode, error_message: "" },
});

const clinic: Clinic = {
	name: "Clínica Sol",
	slug: "sol",
	has_logo: false,
	sections: {
		appointments: true,
		results: true,
		documents: true,
		invoices: true,
	},
	channels: { whatsapp: true, email: true },
	booking: { enabled: true, choose_by: "doctor", new_patients: true },
};

const options: BookingOptions = {
	choose_by: "doctor",
	min_date: "2026-10-27",
	max_date: "2026-12-25",
	doctors: [
		{
			id: 10,
			full_name: "ANA PAZ",
			specialty: "pediatrics",
			specialty_other: "",
			type_ids: [1],
		},
	],
	specialties: [],
	types: [{ id: 1, name: "Consulta", duration_minutes: 30 }],
};

const loadSlots = vi.fn();
const btn = (key: string) => screen.getByRole("button", { name: `*${key}*` });

async function verifyAndPickSlot(user: ReturnType<typeof userEvent.setup>) {
	render(
		<LanguageProvider>
			<AppTextBridge>
				<MemoryRouter>
					<NewPatientFlow clinic={clinic} />
				</MemoryRouter>
			</AppTextBridge>
		</LanguageProvider>,
	);
	await user.type(
		screen.getByLabelText("*portal.new_patient.contact.whatsapp*"),
		"0991234567",
	);
	await user.click(btn("portal.signin.send_code"));
	await user.type(
		await screen.findByLabelText("*portal.signin.code*"),
		"123456",
	);
	await user.click(btn("portal.new_patient.verify"));
	await user.click(await screen.findByRole("button", { name: "09:00" }));
}

async function fillDetails(user: ReturnType<typeof userEvent.setup>) {
	await user.type(
		await screen.findByLabelText("*portal.new_patient.first_name*"),
		"Ana",
	);
	await user.type(
		screen.getByLabelText("*portal.new_patient.last_name*"),
		"Paz",
	);
	await user.type(
		screen.getByLabelText("*portal.new_patient.document.cedula*"),
		"1710034065",
	);
	fireEvent.change(screen.getByLabelText("*portal.new_patient.birth_date*"), {
		target: { value: "1990-05-01" },
	});
	await user.click(btn("portal.new_patient.continue"));
}

describe("NewPatientFlow", () => {
	beforeEach(() => {
		vi.mocked(requestNewPatientCode).mockResolvedValue(ok({ expires_in: 600 }));
		vi.mocked(verifyNewPatientCode).mockResolvedValue(
			ok({ guest_token: "guest-1", expires_in: 1800 }),
		);
		vi.mocked(getBookingOptions).mockResolvedValue(ok(options));
		loadSlots.mockReset();
		loadSlots.mockResolvedValue(
			ok({ days: [{ date: "2026-10-28", times: ["09:00", "09:30"] }] }),
		);
		vi.mocked(guestSlotsLoader).mockReturnValue(loadSlots);
		vi.mocked(createNewPatientRequest).mockReset();
	});

	it("verifies the contact, books with the guest token and sends the data", async () => {
		vi.mocked(createNewPatientRequest).mockResolvedValue(
			ok({
				id: 7,
				status: "pending" as const,
				date: "2026-10-28",
				start_time: "09:00",
				end_time: "09:30",
				doctor: { full_name: "ANA PAZ", specialty: "", specialty_other: "" },
				type_name: "Consulta",
				expires_at: "",
				contact_channel: "whatsapp" as const,
			}),
		);
		const user = userEvent.setup();
		await verifyAndPickSlot(user);

		expect(requestNewPatientCode).toHaveBeenCalledWith("sol", {
			channel: "whatsapp",
			contact: "0991234567",
			website: "",
		});
		expect(verifyNewPatientCode).toHaveBeenCalledWith("sol", {
			channel: "whatsapp",
			contact: "0991234567",
			code: "123456",
		});
		expect(getBookingOptions).toHaveBeenCalledWith("guest-1");
		expect(guestSlotsLoader).toHaveBeenCalledWith("guest-1");

		// The verified phone is filled in and locked.
		expect(
			await screen.findByLabelText("*portal.new_patient.phone*"),
		).toHaveValue("0991234567");
		expect(screen.getByLabelText("*portal.new_patient.phone*")).toHaveAttribute(
			"readonly",
		);
		await fillDetails(user);

		await user.click(
			await screen.findByRole("button", { name: "*portal.booking.submit*" }),
		);
		expect(createNewPatientRequest).toHaveBeenCalledWith("guest-1", {
			type_id: 1,
			date: "2026-10-28",
			start_time: "09:00",
			doctor_id: 10,
			patient: {
				first_name: "Ana",
				last_name: "Paz",
				document_type: "cedula",
				document: "1710034065",
				birth_date: "1990-05-01",
				phone: "0991234567",
				whatsapp_opt_in: false,
			},
		});
		expect(
			await screen.findByRole("heading", {
				name: "*portal.new_patient.done.title*",
			}),
		).toBeInTheDocument();
		expect(
			screen.getByText("*portal.new_patient.done.body.whatsapp*"),
		).toBeInTheDocument();
	});

	it("rejects a wrong cédula before sending", async () => {
		const user = userEvent.setup();
		await verifyAndPickSlot(user);
		await user.type(
			await screen.findByLabelText("*portal.new_patient.first_name*"),
			"Ana",
		);
		await user.type(
			screen.getByLabelText("*portal.new_patient.last_name*"),
			"Paz",
		);
		await user.type(
			screen.getByLabelText("*portal.new_patient.document.cedula*"),
			"1710034066",
		);
		await user.click(btn("portal.new_patient.continue"));
		expect(
			// Field errors show the bare key when it is not translated.
			await screen.findByText("portal.new_patient.error.cedula"),
		).toBeInTheDocument();
	});

	it("says so inline when there is already a pending request", async () => {
		vi.mocked(createNewPatientRequest).mockResolvedValue(
			fail(409, "E-PORTAL-017"),
		);
		const user = userEvent.setup();
		await verifyAndPickSlot(user);
		await fillDetails(user);
		await user.click(
			await screen.findByRole("button", { name: "*portal.booking.submit*" }),
		);
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"*portal.new_patient.error.already_pending*",
		);
	});

	it("on a taken slot goes back to the times and then to the review", async () => {
		vi.mocked(createNewPatientRequest).mockResolvedValue(
			fail(409, "E-PORTAL-010"),
		);
		const user = userEvent.setup();
		await verifyAndPickSlot(user);
		await fillDetails(user);
		await user.click(
			await screen.findByRole("button", { name: "*portal.booking.submit*" }),
		);

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"*portal.new_patient.error.slot_taken*",
		);
		await waitFor(() => expect(loadSlots).toHaveBeenCalledTimes(2));
		await user.click(await screen.findByRole("button", { name: "09:30" }));
		expect(
			await screen.findByRole("heading", {
				name: "*portal.booking.step.review*",
			}),
		).toBeInTheDocument();
	});

	it("starts over when the guest token lapsed", async () => {
		vi.mocked(createNewPatientRequest).mockResolvedValue(
			fail(401, "E-PORTAL-002"),
		);
		const user = userEvent.setup();
		await verifyAndPickSlot(user);
		await fillDetails(user);
		await user.click(
			await screen.findByRole("button", { name: "*portal.booking.submit*" }),
		);
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"*portal.new_patient.error.expired*",
		);
		expect(
			screen.getByLabelText("*portal.new_patient.contact.whatsapp*"),
		).toHaveValue("0991234567");
	});
});
