import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@/test/test-wrapper";

const { userLogin, userLoginTwoFactor, setToken, navigate } = vi.hoisted(
	() => ({
		userLogin: vi.fn(),
		userLoginTwoFactor: vi.fn(),
		setToken: vi.fn(),
		navigate: vi.fn(),
	}),
);

vi.mock("@/api/auth-service", () => ({ userLogin, userLoginTwoFactor }));

vi.mock("react-router", () => ({
	Link: ({ children }: { children: React.ReactNode }) => (
		<a href="/">{children}</a>
	),
	useNavigate: () => navigate,
	useSearchParams: () => [new URLSearchParams(), vi.fn()],
}));

vi.mock("@/store/token-store", () => ({
	useTokenStore: () => ({ setToken }),
}));

import LoginForm from "@/sections/forms/login/login-form";

const MFA_TOKEN = "mfa-token-never-stored";
const session = { token: "tok-2fa", exchange_token: "ex-1", user_id: 7 };

const ok = <T,>(data: T) => ({ success: true, code: 200, message: "", data });
const fail = (code: number, error_code: string) => ({
	success: false,
	code,
	message: "",
	data: { error_code, error_message: "x" },
});

async function submitPassword(user: ReturnType<typeof userEvent.setup>) {
	await user.type(screen.getByLabelText("*login.username*"), "doctor");
	await user.type(screen.getByLabelText("*login.password*"), "password123");
	await user.click(
		screen.getByRole("button", { name: "*login.login_button*" }),
	);
}

async function codeField() {
	const [first] = await screen.findAllByRole("textbox", {
		name: "*login.two_factor.code_label*",
	});
	return first;
}

function storedAnywhere(value: string) {
	for (const storage of [localStorage, sessionStorage]) {
		for (let i = 0; i < storage.length; i++) {
			const key = storage.key(i) ?? "";
			if (key.includes(value) || storage.getItem(key)?.includes(value)) {
				return true;
			}
		}
	}
	return false;
}

describe("LoginForm with two-factor authentication", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		localStorage.clear();
		sessionStorage.clear();
	});

	it("without 2FA signs in right away", async () => {
		userLogin.mockResolvedValue(ok(session));
		const user = userEvent.setup();
		renderWithProviders(<LoginForm />);

		await submitPassword(user);

		await waitFor(() => expect(setToken).toHaveBeenCalledWith("tok-2fa"));
		expect(navigate).toHaveBeenCalledWith(
			"/login/environments?exchange_token=ex-1",
		);
		expect(userLoginTwoFactor).not.toHaveBeenCalled();
	});

	it("asks for the code, then finishes the login like a plain one", async () => {
		userLogin.mockResolvedValue(
			ok({ two_factor_required: true, mfa_token: MFA_TOKEN }),
		);
		userLoginTwoFactor.mockResolvedValue(ok(session));
		const user = userEvent.setup();
		renderWithProviders(<LoginForm />);

		await submitPassword(user);

		expect(
			await screen.findByText("*login.two_factor.title*"),
		).toBeInTheDocument();
		// No session yet, and the mfa token stays in memory only.
		expect(setToken).not.toHaveBeenCalled();
		expect(navigate).not.toHaveBeenCalled();
		expect(storedAnywhere(MFA_TOKEN)).toBe(false);

		await user.type(await codeField(), "123456");
		await user.click(
			screen.getByRole("button", { name: "*login.two_factor.submit*" }),
		);

		await waitFor(() =>
			expect(userLoginTwoFactor).toHaveBeenCalledWith(MFA_TOKEN, "123456"),
		);
		expect(setToken).toHaveBeenCalledWith("tok-2fa");
		expect(navigate).toHaveBeenCalledWith(
			"/login/environments?exchange_token=ex-1",
		);
		expect(storedAnywhere(MFA_TOKEN)).toBe(false);
	});

	it("accepts a recovery code", async () => {
		userLogin.mockResolvedValue(
			ok({ two_factor_required: true, mfa_token: MFA_TOKEN }),
		);
		userLoginTwoFactor.mockResolvedValue(ok(session));
		const user = userEvent.setup();
		renderWithProviders(<LoginForm />);

		await submitPassword(user);
		await user.click(
			await screen.findByRole("button", {
				name: "*login.two_factor.use_recovery*",
			}),
		);
		await user.type(
			screen.getByLabelText("*login.two_factor.recovery_label*"),
			"ABCD-EFGH-IJKL-MNOP",
		);
		await user.click(
			screen.getByRole("button", { name: "*login.two_factor.submit*" }),
		);

		await waitFor(() =>
			expect(userLoginTwoFactor).toHaveBeenCalledWith(
				MFA_TOKEN,
				"ABCD-EFGH-IJKL-MNOP",
			),
		);
		expect(setToken).toHaveBeenCalledWith("tok-2fa");
	});

	it("stays on the code step after a wrong code", async () => {
		userLogin.mockResolvedValue(
			ok({ two_factor_required: true, mfa_token: MFA_TOKEN }),
		);
		userLoginTwoFactor.mockResolvedValue(fail(401, "E-AUTH-019"));
		const user = userEvent.setup();
		renderWithProviders(<LoginForm />);

		await submitPassword(user);
		await user.type(await codeField(), "000000");
		await user.click(
			screen.getByRole("button", { name: "*login.two_factor.submit*" }),
		);

		await waitFor(() => expect(userLoginTwoFactor).toHaveBeenCalled());
		expect(screen.getByText("*login.two_factor.title*")).toBeInTheDocument();
		expect(setToken).not.toHaveBeenCalled();
	});

	it("returns to the password step when the mfa token expired", async () => {
		userLogin.mockResolvedValue(
			ok({ two_factor_required: true, mfa_token: MFA_TOKEN }),
		);
		userLoginTwoFactor.mockResolvedValue(fail(401, "E-AUTH-022"));
		const user = userEvent.setup();
		renderWithProviders(<LoginForm />);

		await submitPassword(user);
		await user.type(await codeField(), "123456");
		await user.click(
			screen.getByRole("button", { name: "*login.two_factor.submit*" }),
		);

		expect(
			await screen.findByRole("button", { name: "*login.login_button*" }),
		).toBeInTheDocument();
		expect(setToken).not.toHaveBeenCalled();
	});

	it("back to login resets to the password step", async () => {
		userLogin.mockResolvedValue(
			ok({ two_factor_required: true, mfa_token: MFA_TOKEN }),
		);
		const user = userEvent.setup();
		renderWithProviders(<LoginForm />);

		await submitPassword(user);
		await user.click(
			await screen.findByRole("button", { name: "*login.two_factor.back*" }),
		);

		expect(
			screen.getByRole("button", { name: "*login.login_button*" }),
		).toBeInTheDocument();
		expect(userLoginTwoFactor).not.toHaveBeenCalled();
	});
});
