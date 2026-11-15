import { AppTextBridge, LanguageProvider } from "@pengi/shared";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import {
	initialPatientValues,
	type VerifiedContact,
} from "@/lib/new-patient-flow";
import { DetailsForm } from "./details-form";

const renderForm = (verified: VerifiedContact) => {
	const onSubmit = vi.fn();
	// The Form components of @pengi/ui read their texts through AppTextBridge.
	render(
		<LanguageProvider>
			<AppTextBridge>
				<DetailsForm
					verified={verified}
					initialValues={initialPatientValues(verified)}
					onSubmit={onSubmit}
				/>
			</AppTextBridge>
		</LanguageProvider>,
	);
	return onSubmit;
};

async function fillRequired(
	user: ReturnType<typeof userEvent.setup>,
	document: string,
) {
	await user.type(
		screen.getByLabelText("*portal.new_patient.first_name*"),
		"Ana",
	);
	await user.type(
		screen.getByLabelText("*portal.new_patient.last_name*"),
		"Paz",
	);
	await user.type(
		screen.getByLabelText("*portal.new_patient.document.cedula*"),
		document,
	);
	fireEvent.change(screen.getByLabelText("*portal.new_patient.birth_date*"), {
		target: { value: "1990-05-01" },
	});
}

describe("DetailsForm", () => {
	it("locks the verified email and submits the person's data", async () => {
		const user = userEvent.setup();
		const onSubmit = renderForm({
			channel: "email",
			contact: "ana@example.com",
		});
		expect(
			screen.getByLabelText(/\*portal\.new_patient\.email\*/),
		).toHaveAttribute("readonly");
		expect(screen.getByLabelText(/\*portal\.new_patient\.email\*/)).toHaveValue(
			"ana@example.com",
		);
		expect(
			screen.getByLabelText(/\*portal\.new_patient\.phone\*/),
		).not.toHaveAttribute("readonly");
		// No phone: no WhatsApp consent to ask for.
		expect(
			screen.queryByText("*portal.new_patient.whatsapp_opt_in*"),
		).toBeNull();

		await fillRequired(user, "1710034065");
		await user.click(
			screen.getByRole("button", { name: /portal\.new_patient\.continue/ }),
		);
		await waitFor(() => expect(onSubmit).toHaveBeenCalled());
		expect(onSubmit.mock.calls[0][0]).toMatchObject({
			first_name: "Ana",
			last_name: "Paz",
			document_type: "cedula",
			document: "1710034065",
			birth_date: "1990-05-01",
			email: "ana@example.com",
		});
	});

	it("asks for WhatsApp consent when the verified contact is a phone", () => {
		renderForm({ channel: "whatsapp", contact: "0991234567" });
		expect(
			screen.getByLabelText(/\*portal\.new_patient\.phone\*/),
		).toHaveAttribute("readonly");
		expect(
			screen.getByText("*portal.new_patient.whatsapp_opt_in*"),
		).toBeVisible();
	});

	it("does not submit a wrong cédula", async () => {
		const user = userEvent.setup();
		const onSubmit = renderForm({
			channel: "email",
			contact: "ana@example.com",
		});
		await fillRequired(user, "0102030405");
		await user.click(
			screen.getByRole("button", { name: /portal\.new_patient\.continue/ }),
		);
		await waitFor(() =>
			expect(
				screen.getByLabelText("*portal.new_patient.document.cedula*"),
			).toHaveAttribute("aria-invalid", "true"),
		);
		expect(onSubmit).not.toHaveBeenCalled();
	});

	it("switches the document label to passport", async () => {
		const user = userEvent.setup();
		renderForm({ channel: "email", contact: "ana@example.com" });
		await user.click(
			screen.getByRole("radio", {
				name: "*portal.new_patient.document_type.passport*",
			}),
		);
		expect(
			screen.getByLabelText("*portal.new_patient.document.passport*"),
		).toBeVisible();
	});
});
