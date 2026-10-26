import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { requestCode, verifyCode } from "@/api/portal-service";
import { SignInForm } from "./sign-in-form";

vi.mock("@/api/portal-service", () => ({
	requestCode: vi.fn(),
	verifyCode: vi.fn(),
}));

// The site key each test runs with (unset: no captcha).
const mockEnv = vi.hoisted(() => ({
	VITE_API_URL: "http://localhost:8000/api/v1",
	VITE_TURNSTILE_SITE_KEY: undefined as string | undefined,
}));
vi.mock("@/lib/env", () => ({ env: mockEnv }));

// The Turnstile widget, faked: "solve" hands over a token, reset() clears it.
const widget = vi.hoisted(() => ({ resets: 0, solved: 0 }));
vi.mock("@/components/turnstile", async () => {
	const React = await import("react");
	return {
		Turnstile: ({
			siteKey,
			onToken,
			ref,
		}: {
			siteKey: string;
			onToken: (token: string | undefined) => void;
			ref?: React.Ref<{ reset: () => void }>;
		}) => {
			React.useImperativeHandle(ref, () => ({
				reset: () => {
					widget.resets++;
					onToken(undefined);
				},
			}));
			return (
				<button
					type="button"
					data-sitekey={siteKey}
					onClick={() => {
						widget.solved++;
						onToken(`token-${widget.solved}`);
					}}
				>
					solve
				</button>
			);
		},
	};
});

// Messages are not loaded in tests: useText renders each key as *key*.
const label = (key: string) => screen.getByLabelText(`*${key}*`);
const button = (key: string) =>
	screen.getByRole("button", { name: `*${key}*` });

describe("SignInForm", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		mockEnv.VITE_TURNSTILE_SITE_KEY = undefined;
		widget.resets = 0;
		widget.solved = 0;
	});

	it("asks for the cédula, then the code, and hands over the token", async () => {
		vi.mocked(requestCode).mockResolvedValue({
			success: true,
			code: 200,
			message: "",
			data: { expires_in: 600 },
		});
		vi.mocked(verifyCode).mockResolvedValue({
			success: true,
			code: 200,
			message: "",
			data: { access_token: "token-1" },
		});
		const onSignedIn = vi.fn();
		const user = userEvent.setup();
		render(<SignInForm slug="clinica" onSignedIn={onSignedIn} />);

		await user.type(label("portal.signin.document"), "1712345678");
		await user.click(button("portal.signin.send_code"));

		expect(requestCode).toHaveBeenCalledWith("clinica", {
			document: "1712345678",
			website: "",
		});
		expect(screen.queryByText("solve")).toBeNull();
		expect(await screen.findByText("*portal.signin.code_sent*")).toBeVisible();

		await user.type(label("portal.signin.code"), "123456");
		await user.click(button("portal.signin.verify"));

		await waitFor(() => expect(onSignedIn).toHaveBeenCalledWith("token-1"));
		expect(verifyCode).toHaveBeenCalledWith("clinica", "1712345678", "123456");
	});

	it("shows a generic error when the code is wrong", async () => {
		vi.mocked(requestCode).mockResolvedValue({
			success: true,
			code: 200,
			message: "",
			data: { expires_in: 600 },
		});
		vi.mocked(verifyCode).mockResolvedValue({
			success: false,
			code: 401,
			message: "x",
			data: { error_code: "E-PORTAL-002", error_message: "x" },
		});
		const user = userEvent.setup();
		render(<SignInForm slug="clinica" onSignedIn={vi.fn()} />);

		await user.type(label("portal.signin.document"), "1712345678");
		await user.click(button("portal.signin.send_code"));
		await user.type(
			await screen.findByLabelText("*portal.signin.code*"),
			"000000",
		);
		await user.click(button("portal.signin.verify"));

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"*portal.signin.code_invalid*",
		);
	});

	it("has a hidden honeypot field named website", () => {
		const { container } = render(
			<SignInForm slug="clinica" onSignedIn={vi.fn()} />,
		);
		const honeypot = container.querySelector<HTMLInputElement>(
			'input[name="website"]',
		);

		expect(honeypot).not.toBeNull();
		expect(honeypot).toHaveAttribute("tabindex", "-1");
		expect(honeypot).toHaveAttribute("autocomplete", "off");
		expect(honeypot?.closest("[aria-hidden='true']")).not.toBeNull();
	});

	it("sends what a bot typed in the honeypot", async () => {
		vi.mocked(requestCode).mockResolvedValue({
			success: true,
			code: 200,
			message: "",
			data: { expires_in: 600 },
		});
		const user = userEvent.setup();
		const { container } = render(
			<SignInForm slug="clinica" onSignedIn={vi.fn()} />,
		);
		const honeypot = container.querySelector<HTMLInputElement>(
			'input[name="website"]',
		) as HTMLInputElement;

		await user.type(label("portal.signin.document"), "1712345678");
		await user.type(honeypot, "spam.example");
		await user.click(button("portal.signin.send_code"));

		expect(requestCode).toHaveBeenCalledWith("clinica", {
			document: "1712345678",
			website: "spam.example",
		});
	});

	it("with a site key, waits for the captcha and sends its token", async () => {
		mockEnv.VITE_TURNSTILE_SITE_KEY = "site-key";
		vi.mocked(requestCode).mockResolvedValue({
			success: false,
			code: 400,
			message: "x",
			data: { error_code: "E-PORTAL-008", error_message: "x" },
		});
		const user = userEvent.setup();
		render(<SignInForm slug="clinica" onSignedIn={vi.fn()} />);

		await user.type(label("portal.signin.document"), "1712345678");
		expect(button("portal.signin.send_code")).toBeDisabled();
		expect(screen.getByText("solve")).toHaveAttribute(
			"data-sitekey",
			"site-key",
		);

		await user.click(screen.getByText("solve"));
		expect(button("portal.signin.send_code")).toBeEnabled();
		await user.click(button("portal.signin.send_code"));

		expect(requestCode).toHaveBeenCalledWith("clinica", {
			document: "1712345678",
			website: "",
			captcha_token: "token-1",
		});
		// The token was spent: the widget is reset and the button waits again.
		await waitFor(() => expect(widget.resets).toBe(1));
		expect(button("portal.signin.send_code")).toBeDisabled();

		await user.click(screen.getByText("solve"));
		await user.click(button("portal.signin.send_code"));
		expect(requestCode).toHaveBeenLastCalledWith("clinica", {
			document: "1712345678",
			website: "",
			captcha_token: "token-2",
		});
	});
});
