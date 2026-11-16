import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@/test/test-wrapper";

const mocks = vi.hoisted(() => ({
	beginPasskeyLogin: vi.fn(),
	loginWithPasskey: vi.fn(),
	startAuthentication: vi.fn(),
	browserSupportsWebAuthn: vi.fn(),
	setToken: vi.fn(),
	navigate: vi.fn(),
}));

vi.mock("@/api/auth-service", () => ({
	userLogin: vi.fn(),
	userLoginTwoFactor: vi.fn(),
}));
vi.mock("@/api/passkey-service", () => ({
	beginPasskeyLogin: mocks.beginPasskeyLogin,
	loginWithPasskey: mocks.loginWithPasskey,
}));
vi.mock("@simplewebauthn/browser", () => ({
	startAuthentication: mocks.startAuthentication,
	browserSupportsWebAuthn: mocks.browserSupportsWebAuthn,
}));
vi.mock("react-router", () => ({
	Link: ({ children }: { children: React.ReactNode }) => (
		<a href="/">{children}</a>
	),
	useNavigate: () => mocks.navigate,
	useSearchParams: () => [new URLSearchParams(), vi.fn()],
}));
vi.mock("@/store/token-store", () => ({
	useTokenStore: () => ({ setToken: mocks.setToken }),
}));

import LoginForm from "@/sections/forms/login/login-form";

const ok = <T,>(data: T) => ({ success: true, code: 200, message: "", data });
const session = { token: "tok-pk", exchange_token: "ex-9", user_id: 3 };
const BUTTON = { name: "*auth.passkey.login_button*" };

describe("LoginForm passkey sign-in", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		mocks.browserSupportsWebAuthn.mockReturnValue(true);
		mocks.beginPasskeyLogin.mockResolvedValue(
			ok({ challenge_id: "c1", options: { challenge: "abc" } }),
		);
	});

	it("runs begin -> startAuthentication -> login and completes the session", async () => {
		mocks.startAuthentication.mockResolvedValue({ id: "cred" });
		mocks.loginWithPasskey.mockResolvedValue(ok(session));
		const user = userEvent.setup();
		renderWithProviders(<LoginForm />);

		await user.click(screen.getByRole("button", BUTTON));

		await waitFor(() =>
			expect(mocks.loginWithPasskey).toHaveBeenCalledWith("c1", {
				id: "cred",
			}),
		);
		expect(mocks.startAuthentication).toHaveBeenCalledWith({
			optionsJSON: { challenge: "abc" },
		});
		expect(mocks.setToken).toHaveBeenCalledWith("tok-pk");
		expect(mocks.navigate).toHaveBeenCalledWith(
			"/login/environments?exchange_token=ex-9",
		);
	});

	it("a cancelled prompt does not log in or toast", async () => {
		mocks.startAuthentication.mockRejectedValue(
			Object.assign(new Error("cancelled"), { name: "NotAllowedError" }),
		);
		const user = userEvent.setup();
		renderWithProviders(<LoginForm />);

		await user.click(screen.getByRole("button", BUTTON));

		await waitFor(() => expect(mocks.startAuthentication).toHaveBeenCalled());
		expect(mocks.loginWithPasskey).not.toHaveBeenCalled();
		expect(mocks.setToken).not.toHaveBeenCalled();
		expect(screen.queryByText("*auth.passkey.browser_error*")).toBeNull();
	});

	it("is hidden when the browser lacks WebAuthn", () => {
		mocks.browserSupportsWebAuthn.mockReturnValue(false);
		renderWithProviders(<LoginForm />);

		expect(screen.queryByRole("button", BUTTON)).toBeNull();
	});
});
