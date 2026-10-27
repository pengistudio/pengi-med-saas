import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { AppointmentRequest } from "@/api/appointment-request-service";
import { RequesterCell } from "@/components/features/appointment-requests/requester-cell";
import { renderWithProviders } from "@/test/test-wrapper";

const base: AppointmentRequest = {
	id: 1,
	status: "pending",
	patient: null,
	doctor: { id: 3, full_name: "Ana Paz", specialty: "", specialty_other: "" },
	type_name: "Consulta",
	date: "2026-10-28",
	start_time: "09:00",
	end_time: "09:30",
	reason: "",
	expires_at: "2026-10-27T09:00:00Z",
	created_at: "2026-10-26T09:00:00Z",
	new_patient: {
		first_name: "Luis",
		last_name: "Mora",
		document_type: "cedula",
		document: "1710034065",
		birth_date: "1990-05-01",
		phone: "+593991234567",
		email: "",
		contact_channel: "whatsapp",
	},
	matching_patient: null,
};

// Messages are not loaded in tests: useText renders each key as *key*.
describe("RequesterCell", () => {
	it("marks a new patient and says the record will be created", () => {
		renderWithProviders(<RequesterCell request={base} />);
		expect(screen.getByText("Luis Mora")).toBeDefined();
		expect(
			screen.getByText("*appointment_requests.new_patient.badge*"),
		).toBeDefined();
		expect(screen.getByText("1710034065")).toBeDefined();
		expect(screen.getByText("+593991234567")).toBeDefined();
		expect(
			screen.getByText("*appointment_requests.new_patient.verified.whatsapp*"),
		).toBeDefined();
		expect(
			screen.getByText("*appointment_requests.new_patient.will_create*"),
		).toBeDefined();
	});

	it("says which record approving links to", () => {
		renderWithProviders(
			<RequesterCell
				request={{
					...base,
					matching_patient: { id: 9, first_name: "Luis", last_name: "Mora" },
				}}
			/>,
		);
		expect(
			screen.getByText("*appointment_requests.new_patient.matching*"),
		).toBeDefined();
	});

	it("shows a patient's own request as before", () => {
		renderWithProviders(
			<RequesterCell
				request={{
					...base,
					new_patient: null,
					patient: {
						id: 2,
						first_name: "Eva",
						last_name: "Ruiz",
						document: "0102030400",
						phone: "0991",
					},
				}}
			/>,
		);
		expect(screen.getByText("Eva Ruiz")).toBeDefined();
		expect(screen.getByText("0102030400 · 0991")).toBeDefined();
		expect(
			screen.queryByText("*appointment_requests.new_patient.badge*"),
		).toBeNull();
	});
});
