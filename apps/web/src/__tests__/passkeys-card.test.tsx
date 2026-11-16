import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@/test/test-wrapper";

const mocks = vi.hoisted(() => ({
	getPasskeys: vi.fn(),
	beginPasskeyRegistration: vi.fn(),
	finishPasskeyRegistration: vi.fn(),
	deletePasskey: vi.fn(),
	getTwoFactorStatus: vi.fn(),
	startRegistration: vi.fn(),
	browserSupportsWebAuthn: vi.fn(),
}));

vi.mock("@/api/passkey-service", () => ({
	getPasskeys: mocks.getPasskeys,
	beginPasskeyRegistration: mocks.beginPasskeyRegistration,
	finishPasskeyRegistration: mocks.finishPasskeyRegistration,
	deletePasskey: mocks.deletePasskey,
}));
vi.mock("@/api/two-factor-service", () => ({
	getTwoFactorStatus: mocks.getTwoFactorStatus,
}));
vi.mock("@simplewebauthn/browser", () => ({
	startRegistration: mocks.startRegistration,
	browserSupportsWebAuthn: mocks.browserSupportsWebAuthn,
}));

import { PasskeysCard } from "@/components/features/profile/passkeys-card";

const ok = <T,>(data: T) => ({ success: true, code: 200, message: "", data });
const PASSKEYS = [
	{
		id: 1,
		name: "MacBook",
		created_at: "2026-01-01T10:00:00Z",
		last_used_at: null,
	},
];

async function openAdd(user: ReturnType<typeof userEvent.setup>) {
	await screen.findByText("MacBook");
	await user.click(
		screen.getByRole("button", { name: "*profile.passkeys.add*" }),
	);
}

async function fillAndSubmit(
	user: ReturnType<typeof userEvent.setup>,
	code?: string,
) {
	await user.type(
		await screen.findByLabelText("*profile.passkeys.name_label*"),
		"Phone",
	);
	await user.type(
		screen.getByLabelText("*profile.security.current_password*"),
		"password123",
	);
	if (code) {
		await user.type(
			screen.getByLabelText("*profile.two_factor.proof.code_label*"),
			code,
		);
	}
	await user.click(
		screen.getByRole("button", { name: "*profile.passkeys.add.submit*" }),
	);
}

describe("PasskeysCard", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		mocks.browserSupportsWebAuthn.mockReturnValue(true);
		mocks.getPasskeys.mockResolvedValue(ok(PASSKEYS));
		mocks.getTwoFactorStatus.mockResolvedValue(
			ok({ enabled: false, recovery_codes_remaining: 0 }),
		);
		mocks.beginPasskeyRegistration.mockResolvedValue(
			ok({ challenge_id: "c1", options: { challenge: "abc" } }),
		);
		mocks.startRegistration.mockResolvedValue({ id: "cred" });
		mocks.finishPasskeyRegistration.mockResolvedValue(ok(PASSKEYS[0]));
	});

	it("lists passkeys with never-used state", async () => {
		renderWithProviders(<PasskeysCard />);

		expect(await screen.findByText("MacBook")).toBeInTheDocument();
		expect(
			screen.getByText(/\*profile\.passkeys\.never_used\*/),
		).toBeInTheDocument();
	});

	it("adds a passkey without a code when 2FA is off", async () => {
		const user = userEvent.setup();
		renderWithProviders(<PasskeysCard />);

		await openAdd(user);
		await screen.findByLabelText("*profile.passkeys.name_label*");
		expect(
			screen.queryByLabelText("*profile.two_factor.proof.code_label*"),
		).toBeNull();
		await fillAndSubmit(user);

		await waitFor(() =>
			expect(mocks.finishPasskeyRegistration).toHaveBeenCalledWith(
				"c1",
				"Phone",
				{ id: "cred" },
			),
		);
		expect(mocks.beginPasskeyRegistration).toHaveBeenCalledWith(
			"password123",
			undefined,
		);
		expect(mocks.startRegistration).toHaveBeenCalledWith({
			optionsJSON: { challenge: "abc" },
		});
	});

	it("asks for the code only when 2FA is on", async () => {
		mocks.getTwoFactorStatus.mockResolvedValue(
			ok({ enabled: true, recovery_codes_remaining: 8 }),
		);
		const user = userEvent.setup();
		renderWithProviders(<PasskeysCard />);

		await openAdd(user);
		await fillAndSubmit(user, "123456");

		await waitFor(() =>
			expect(mocks.beginPasskeyRegistration).toHaveBeenCalledWith(
				"password123",
				"123456",
			),
		);
	});

	it("a cancelled prompt finishes nothing and shows no error", async () => {
		mocks.startRegistration.mockRejectedValue(
			Object.assign(new Error("cancelled"), { name: "NotAllowedError" }),
		);
		const user = userEvent.setup();
		renderWithProviders(<PasskeysCard />);

		await openAdd(user);
		await fillAndSubmit(user);

		await waitFor(() => expect(mocks.startRegistration).toHaveBeenCalled());
		expect(mocks.finishPasskeyRegistration).not.toHaveBeenCalled();
		expect(screen.queryByText("*auth.passkey.browser_error*")).toBeNull();
	});

	it("deletes after confirming and reloads", async () => {
		mocks.deletePasskey.mockResolvedValue(ok(null));
		const user = userEvent.setup();
		renderWithProviders(<PasskeysCard />);

		await screen.findByText("MacBook");
		await user.click(
			screen.getByRole("button", { name: "*profile.passkeys.delete*" }),
		);
		const dialog = await screen.findByRole("alertdialog");
		await user.click(
			within(dialog).getByRole("button", {
				name: "*profile.passkeys.delete.confirm*",
			}),
		);

		await waitFor(() => expect(mocks.deletePasskey).toHaveBeenCalledWith(1));
		await waitFor(() => expect(mocks.getPasskeys).toHaveBeenCalledTimes(2));
	});

	it("disables adding when WebAuthn is unsupported", async () => {
		mocks.browserSupportsWebAuthn.mockReturnValue(false);
		renderWithProviders(<PasskeysCard />);

		await screen.findByText("MacBook");
		expect(
			screen.getByRole("button", { name: "*profile.passkeys.add*" }),
		).toBeDisabled();
	});
});
