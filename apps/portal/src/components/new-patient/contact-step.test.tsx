import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	requestNewPatientCode,
	verifyNewPatientCode,
} from "@/api/portal-service";
import { findCodeField } from "@/test/code-field";
import { CodeForm, ContactForm } from "./contact-step";

vi.mock("@/api/portal-service", () => ({
	NEW_PATIENT_ERROR: {
		invalidContact: "E-PORTAL-005",
		captcha: "E-PORTAL-008",
		notAccepted: "E-PORTAL-015",
	},
	requestNewPatientCode: vi.fn(),
	verifyNewPatientCode: vi.fn(),
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
const fail = (code: number, errorCode: string, message = "server message") => ({
	success: false as const,
	code,
	message,
	data: { error_code: errorCode, error_message: "" },
});

const btn = (key: string) => screen.getByRole("button", { name: `*${key}*` });

describe("ContactForm", () => {
	const onCodeSent = vi.fn();
	const onNotAccepted = vi.fn();
	const renderForm = (
		channels: ("whatsapp" | "email")[] = ["whatsapp", "email"],
	) =>
		render(
			<ContactForm
				slug="sol"
				channels={channels}
				onCodeSent={onCodeSent}
				onNotAccepted={onNotAccepted}
			/>,
		);

	beforeEach(() => vi.clearAllMocks());

	it("sends the code to the chosen channel's contact, trimmed", async () => {
		vi.mocked(requestNewPatientCode).mockResolvedValue(ok({ expires_in: 600 }));
		const user = userEvent.setup();
		renderForm();
		await user.click(
			screen.getByLabelText("*portal.new_patient.channel.email*"),
		);
		await user.type(
			screen.getByLabelText("*portal.new_patient.contact.email*"),
			"  ana@example.com ",
		);
		await user.click(btn("portal.signin.send_code"));
		expect(requestNewPatientCode).toHaveBeenCalledWith("sol", {
			channel: "email",
			contact: "ana@example.com",
			website: "",
		});
		expect(onCodeSent).toHaveBeenCalledWith({
			channel: "email",
			contact: "ana@example.com",
		});
	});

	it("rejects a malformed contact without calling the API", async () => {
		const user = userEvent.setup();
		renderForm(["email"]);
		expect(screen.queryByText("*portal.new_patient.channel*")).toBeNull();
		await user.type(
			screen.getByLabelText("*portal.new_patient.contact.email*"),
			"not-an-email",
		);
		await user.click(btn("portal.signin.send_code"));
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"*portal.new_patient.error.contact_email*",
		);
		expect(requestNewPatientCode).not.toHaveBeenCalled();
	});

	it("shows the captcha error inline and hands a closed clinic to the page", async () => {
		const user = userEvent.setup();
		renderForm();
		await user.type(
			screen.getByLabelText("*portal.new_patient.contact.whatsapp*"),
			"0991234567",
		);
		vi.mocked(requestNewPatientCode).mockResolvedValueOnce(
			fail(400, "E-PORTAL-008"),
		);
		await user.click(btn("portal.signin.send_code"));
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"*portal.signin.captcha_error*",
		);

		vi.mocked(requestNewPatientCode).mockResolvedValueOnce(
			fail(404, "E-PORTAL-015"),
		);
		await user.click(btn("portal.signin.send_code"));
		expect(onNotAccepted).toHaveBeenCalled();
		expect(onCodeSent).not.toHaveBeenCalled();
	});
});

describe("CodeForm", () => {
	const onVerified = vi.fn();
	const onChangeContact = vi.fn();
	const contact = { channel: "whatsapp" as const, contact: "593991234567" };

	beforeEach(() => vi.clearAllMocks());

	it("trades a right code for the guest token", async () => {
		vi.mocked(verifyNewPatientCode).mockResolvedValue(
			ok({ guest_token: "guest-1", expires_in: 1800 }),
		);
		const user = userEvent.setup();
		render(
			<CodeForm
				slug="sol"
				contact={contact}
				onVerified={onVerified}
				onChangeContact={onChangeContact}
			/>,
		);
		expect(btn("portal.new_patient.verify")).toBeDisabled();
		await user.type(await findCodeField(), "123456");
		await user.click(btn("portal.new_patient.verify"));
		expect(verifyNewPatientCode).toHaveBeenCalledWith("sol", {
			...contact,
			code: "123456",
		});
		expect(onVerified).toHaveBeenCalledWith("guest-1");
	});

	it("says the code is wrong on a 401 and lets the person change the contact", async () => {
		vi.mocked(verifyNewPatientCode).mockResolvedValue(
			fail(401, "E-PORTAL-002"),
		);
		const user = userEvent.setup();
		render(
			<CodeForm
				slug="sol"
				contact={contact}
				onVerified={onVerified}
				onChangeContact={onChangeContact}
			/>,
		);
		await user.type(await findCodeField(), "000000");
		await user.click(btn("portal.new_patient.verify"));
		expect(
			await screen.findByText("*portal.signin.code_invalid*"),
		).toBeVisible();
		expect(onVerified).not.toHaveBeenCalled();
		await user.click(btn("portal.new_patient.change_contact"));
		expect(onChangeContact).toHaveBeenCalled();
	});
});
