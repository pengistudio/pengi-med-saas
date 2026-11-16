import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useTokenStore } from "@/store/token-store";
import { renderWithProviders } from "@/test/test-wrapper";

const service = vi.hoisted(() => ({
	getTwoFactorStatus: vi.fn(),
	setupTwoFactor: vi.fn(),
	enableTwoFactor: vi.fn(),
	disableTwoFactor: vi.fn(),
	regenerateRecoveryCodes: vi.fn(),
}));

vi.mock("@/api/two-factor-service", () => service);

import { TwoFactorCard } from "@/components/features/profile/two-factor-card";

const ok = <T,>(data: T) => ({ success: true, code: 200, message: "", data });
const fail = (code: number, error_code: string) => ({
	success: false,
	code,
	message: "",
	data: { error_code, error_message: "x" },
});

const RECOVERY_CODES = Array.from(
	{ length: 10 },
	(_, i) => `AAAA-BBBB-CCCC-${String(i).padStart(4, "0")}`,
);

async function startEnrollment(user: ReturnType<typeof userEvent.setup>) {
	await user.click(
		await screen.findByRole("button", { name: "*profile.two_factor.enable*" }),
	);
	await user.type(
		await screen.findByLabelText("*profile.security.current_password*"),
		"password123",
	);
	await user.click(
		screen.getByRole("button", { name: "*profile.two_factor.continue*" }),
	);
	const [first] = await screen.findAllByRole("textbox", {
		name: "*profile.two_factor.code_label*",
	});
	return first;
}

describe("TwoFactorCard enrollment", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		useTokenStore.getState().setToken("old-token");
		service.getTwoFactorStatus.mockResolvedValue(
			ok({ enabled: false, recovery_codes_remaining: 0 }),
		);
		service.setupTwoFactor.mockResolvedValue(
			ok({
				secret: "JBSWY3DPEHPK3PXP",
				otpauth_uri:
					"otpauth://totp/Pengi:doctor?secret=JBSWY3DPEHPK3PXP&issuer=Pengi",
			}),
		);
	});

	it("enables 2FA, switches to the new session and shows the codes once", async () => {
		service.enableTwoFactor.mockResolvedValue(
			ok({
				token: "new-token",
				exchange_token: "ex",
				user_id: 7,
				recovery_codes: RECOVERY_CODES,
			}),
		);
		const user = userEvent.setup();
		renderWithProviders(<TwoFactorCard />);

		const code = await startEnrollment(user);
		expect(service.setupTwoFactor).toHaveBeenCalledWith("password123");
		// The secret for manual entry, grouped for reading.
		expect(screen.getByText("JBSW Y3DP EHPK 3PXP")).toBeInTheDocument();

		await user.type(code, "123456");
		await user.click(
			screen.getByRole("button", { name: "*profile.two_factor.verify*" }),
		);

		await waitFor(() =>
			expect(service.enableTwoFactor).toHaveBeenCalledWith("123456"),
		);
		// Every earlier token was revoked: the client carries on with the new one.
		expect(useTokenStore.getState().token).toBe("new-token");

		for (const c of RECOVERY_CODES) {
			expect(await screen.findByText(c)).toBeInTheDocument();
		}
		const done = screen.getByRole("button", {
			name: "*profile.two_factor.codes.done*",
		});
		expect(done).toBeDisabled();
		await user.click(
			screen.getByRole("checkbox", {
				name: "*profile.two_factor.codes.confirm*",
			}),
		);
		expect(done).toBeEnabled();
		await user.click(done);

		await waitFor(() =>
			expect(screen.queryByText(RECOVERY_CODES[0])).not.toBeInTheDocument(),
		);
		// Status reloaded after enabling.
		expect(service.getTwoFactorStatus).toHaveBeenCalledTimes(2);
	});

	it("keeps the session and the step on a wrong code", async () => {
		service.enableTwoFactor.mockResolvedValue(fail(422, "E-AUTH-019"));
		const user = userEvent.setup();
		renderWithProviders(<TwoFactorCard />);

		const code = await startEnrollment(user);
		await user.type(code, "000000");
		await user.click(
			screen.getByRole("button", { name: "*profile.two_factor.verify*" }),
		);

		await waitFor(() => expect(service.enableTwoFactor).toHaveBeenCalled());
		expect(useTokenStore.getState().token).toBe("old-token");
		expect(screen.getByText("JBSW Y3DP EHPK 3PXP")).toBeInTheDocument();
	});

	it("does not go past the password on a wrong password", async () => {
		service.setupTwoFactor.mockResolvedValue(fail(422, "E-AUTH-016"));
		const user = userEvent.setup();
		renderWithProviders(<TwoFactorCard />);

		await user.click(
			await screen.findByRole("button", {
				name: "*profile.two_factor.enable*",
			}),
		);
		await user.type(
			await screen.findByLabelText("*profile.security.current_password*"),
			"wrong",
		);
		await user.click(
			screen.getByRole("button", { name: "*profile.two_factor.continue*" }),
		);

		await waitFor(() => expect(service.setupTwoFactor).toHaveBeenCalled());
		expect(
			screen.queryByText("*profile.two_factor.code_label*"),
		).not.toBeInTheDocument();
		expect(useTokenStore.getState().token).toBe("old-token");
	});

	it("shows the status and recovery codes left when enabled", async () => {
		service.getTwoFactorStatus.mockResolvedValue(
			ok({ enabled: true, recovery_codes_remaining: 2 }),
		);
		renderWithProviders(<TwoFactorCard />);

		expect(
			await screen.findByText("*profile.two_factor.status.enabled*"),
		).toBeInTheDocument();
		expect(screen.getByRole("alert")).toHaveTextContent(
			"*profile.two_factor.recovery_low*",
		);
		expect(
			screen.getByRole("button", { name: "*profile.two_factor.disable*" }),
		).toBeInTheDocument();
	});
});
