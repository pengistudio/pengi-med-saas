import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@/test/test-wrapper";

const { loginTwoFactor } = vi.hoisted(() => ({ loginTwoFactor: vi.fn() }));

vi.mock("@/api/auth-service", () => ({
	loginTwoFactor,
	INVALID_MFA_TOKEN: "E-AUTH-022",
	beginPasskeyLogin: vi.fn(),
	loginWithPasskey: vi.fn(),
}));

import { TwoFactorStep } from "./two-factor-step";

const fail = (error_code: string) => ({
	success: false,
	code: 401,
	message: "",
	data: { error_code, error_message: "x" },
});

function setup(props: { totp?: boolean; passkey?: boolean } = {}) {
	const handlers = {
		onSuccess: vi.fn(),
		onBack: vi.fn(),
		onExpired: vi.fn(),
	};
	renderWithProviders(
		<TwoFactorStep
			mfaToken="mfa"
			totp={props.totp ?? true}
			passkey={props.passkey ?? false}
			{...handlers}
		/>,
	);
	return handlers;
}

describe("TwoFactorStep", () => {
	beforeEach(() => vi.clearAllMocks());

	it("signs in with a recovery code", async () => {
		loginTwoFactor.mockResolvedValue({
			success: true,
			code: 200,
			message: "",
			data: { token: "tok", user_id: 1 },
		});
		const { onSuccess } = setup();
		const user = userEvent.setup();
		await user.click(
			screen.getByRole("button", { name: "*login.two_factor.use_recovery*" }),
		);
		await user.type(
			screen.getByLabelText("*login.two_factor.recovery_label*"),
			"AAAA-BBBB-CCCC-DDDD",
		);
		await user.click(
			screen.getByRole("button", { name: "*login.two_factor.submit*" }),
		);
		await waitFor(() => expect(onSuccess).toHaveBeenCalledWith("tok"));
		expect(loginTwoFactor).toHaveBeenCalledWith("mfa", "AAAA-BBBB-CCCC-DDDD");
	});

	it("goes back to the password step when the token expired", async () => {
		loginTwoFactor.mockResolvedValue(fail("E-AUTH-022"));
		const { onExpired } = setup();
		const user = userEvent.setup();
		await user.click(
			screen.getByRole("button", { name: "*login.two_factor.use_recovery*" }),
		);
		await user.type(
			screen.getByLabelText("*login.two_factor.recovery_label*"),
			"AAAA-BBBB-CCCC-DDDD",
		);
		await user.click(
			screen.getByRole("button", { name: "*login.two_factor.submit*" }),
		);
		await waitFor(() => expect(onExpired).toHaveBeenCalled());
	});

	it("keeps the admin on the step after a wrong code", async () => {
		loginTwoFactor.mockResolvedValue(fail("E-AUTH-019"));
		const { onExpired, onSuccess } = setup();
		const user = userEvent.setup();
		await user.click(
			screen.getByRole("button", { name: "*login.two_factor.use_recovery*" }),
		);
		await user.type(
			screen.getByLabelText("*login.two_factor.recovery_label*"),
			"AAAA-BBBB-CCCC-DDDD",
		);
		await user.click(
			screen.getByRole("button", { name: "*login.two_factor.submit*" }),
		);
		await waitFor(() => expect(loginTwoFactor).toHaveBeenCalled());
		expect(onExpired).not.toHaveBeenCalled();
		expect(onSuccess).not.toHaveBeenCalled();
	});

	it("offers only the passkey to an admin without an authenticator", () => {
		setup({ totp: false, passkey: true });
		expect(
			screen.queryByRole("button", {
				name: "*login.two_factor.use_recovery*",
			}),
		).toBeNull();
	});
});
