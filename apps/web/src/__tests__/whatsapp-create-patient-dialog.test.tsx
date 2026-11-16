import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createWhatsAppConversationPatient } from "@/api/whatsapp-service";
import { CreatePatientDialog } from "@/sections/whatsapp/create-patient-dialog";
import { renderWithProviders } from "@/test/test-wrapper";

vi.mock("@/api/whatsapp-service", async (importOriginal) => {
	const actual =
		await importOriginal<typeof import("@/api/whatsapp-service")>();
	return {
		...actual,
		createWhatsAppConversationPatient: vi.fn(async () => ({
			success: true,
			data: { id: 7, patient_id: 99 },
		})),
	};
});

const createMock = vi.mocked(createWhatsAppConversationPatient);

function renderDialog(defaultName?: string) {
	const onCreated = vi.fn();
	const onOpenChange = vi.fn();
	renderWithProviders(
		<CreatePatientDialog
			open
			onOpenChange={onOpenChange}
			conversationId={7}
			phone="099 888 7777"
			defaultName={defaultName}
			onCreated={onCreated}
		/>,
	);
	return { onCreated, onOpenChange };
}

const submit = () =>
	screen.getByRole("button", {
		name: "*whatsapp.inbox.create_patient.submit*",
	});

describe("CreatePatientDialog", () => {
	beforeEach(() => {
		createMock.mockClear();
	});

	it("does not call the API without a first name", async () => {
		const user = userEvent.setup();
		const { onCreated } = renderDialog();
		await user.click(submit());
		await waitFor(() =>
			expect(
				screen.getAllByText(
					/whatsapp\.inbox\.create_patient\.error\.first_name/,
				).length,
			).toBeGreaterThan(0),
		);
		expect(createMock).not.toHaveBeenCalled();
		expect(onCreated).not.toHaveBeenCalled();
	});

	it("creates the patient for the conversation and closes", async () => {
		const user = userEvent.setup();
		const { onCreated, onOpenChange } = renderDialog();
		const [first, last] = screen.getAllByRole("textbox");
		await user.type(first, "  Carla ");
		await user.type(last, "Mora");
		await user.click(submit());
		await waitFor(() => expect(createMock).toHaveBeenCalledTimes(1));
		expect(createMock).toHaveBeenCalledWith(7, {
			first_name: "Carla",
			last_name: "Mora",
			document: undefined,
		});
		await waitFor(() =>
			expect(onCreated).toHaveBeenCalledWith({ id: 7, patient_id: 99 }),
		);
		expect(onOpenChange).toHaveBeenCalledWith(false);
	});

	it("pre-fills the first name when one is known", () => {
		renderDialog("Carla");
		expect(screen.getByDisplayValue("Carla")).toBeDefined();
	});
});
