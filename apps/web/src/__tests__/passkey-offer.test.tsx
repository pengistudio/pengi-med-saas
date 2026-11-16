import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@/test/test-wrapper";

const mocks = vi.hoisted(() => ({
	getPasskeys: vi.fn(),
	beginPasskeyRegistration: vi.fn(),
	finishPasskeyRegistration: vi.fn(),
	getTwoFactorStatus: vi.fn(),
	startRegistration: vi.fn(),
	browserSupportsWebAuthn: vi.fn(),
	errorToast: vi.fn(),
}));

vi.mock("@/api/passkey-service", () => ({
	PASSKEY_REAUTH_REQUIRED: "E-AUTH-027",
	getPasskeys: mocks.getPasskeys,
	beginPasskeyRegistration: mocks.beginPasskeyRegistration,
	finishPasskeyRegistration: mocks.finishPasskeyRegistration,
}));
vi.mock("@/api/two-factor-service", () => ({
	getTwoFactorStatus: mocks.getTwoFactorStatus,
}));
vi.mock("@simplewebauthn/browser", () => ({
	startRegistration: mocks.startRegistration,
	browserSupportsWebAuthn: mocks.browserSupportsWebAuthn,
}));
vi.mock("@pengi/ui", async (importOriginal) => ({
	...(await importOriginal<typeof import("@pengi/ui")>()),
	useToast: () => ({ errorToast: mocks.errorToast }),
}));

import { PasskeyOffer } from "@/components/features/passkeys/passkey-offer";
import {
	markPasskeyOfferPending,
	takePasskeyOfferPending,
} from "@/lib/passkey-offer-storage";
import { useSessionStore } from "@/store/session-store";

const ok = <T,>(data: T) => ({ success: true, code: 200, message: "", data });
const reauthError = {
	success: false,
	code: 422,
	message: "auth.passkey.reauth_required",
	data: { error_code: "E-AUTH-027", error_message: "" },
};
const TITLE = "*auth.passkey_offer.title*";
const CREATE = { name: "*auth.passkey_offer.create*" };
const USER_ID = 7;

describe("PasskeyOffer", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		localStorage.clear();
		sessionStorage.clear();
		useSessionStore.setState({ mfaRequired: false });
		markPasskeyOfferPending(USER_ID);
		mocks.browserSupportsWebAuthn.mockReturnValue(true);
		mocks.getPasskeys.mockResolvedValue(ok([]));
		mocks.getTwoFactorStatus.mockResolvedValue(
			ok({ enabled: false, recovery_codes_remaining: 0 }),
		);
		mocks.beginPasskeyRegistration.mockResolvedValue(
			ok({ challenge_id: "c1", options: { challenge: "abc" } }),
		);
		mocks.startRegistration.mockResolvedValue({ id: "cred" });
		mocks.finishPasskeyRegistration.mockResolvedValue(ok({ id: 1 }));
	});

	it("shows after a password login when eligible", async () => {
		renderWithProviders(<PasskeyOffer />);

		expect(await screen.findByText(TITLE)).toBeInTheDocument();
		expect(takePasskeyOfferPending()).toBeNull();
	});

	it("stays hidden without the password-login mark (passkey login)", async () => {
		sessionStorage.clear();
		renderWithProviders(<PasskeyOffer />);

		await new Promise((r) => setTimeout(r, 20));
		expect(mocks.getPasskeys).not.toHaveBeenCalled();
		expect(screen.queryByText(TITLE)).toBeNull();
	});

	it("stays hidden when WebAuthn is unsupported", async () => {
		mocks.browserSupportsWebAuthn.mockReturnValue(false);
		renderWithProviders(<PasskeyOffer />);

		await new Promise((r) => setTimeout(r, 20));
		expect(mocks.getPasskeys).not.toHaveBeenCalled();
		expect(screen.queryByText(TITLE)).toBeNull();
	});

	it("stays hidden when the user already has passkeys", async () => {
		mocks.getPasskeys.mockResolvedValue(ok([{ id: 1 }]));
		renderWithProviders(<PasskeyOffer />);

		await waitFor(() => expect(mocks.getPasskeys).toHaveBeenCalled());
		await new Promise((r) => setTimeout(r, 20));
		expect(screen.queryByText(TITLE)).toBeNull();
	});

	it("stays hidden while the MFA-required redirect is active", async () => {
		useSessionStore.setState({ mfaRequired: true });
		renderWithProviders(<PasskeyOffer />);

		await waitFor(() => expect(mocks.getPasskeys).toHaveBeenCalled());
		await new Promise((r) => setTimeout(r, 20));
		expect(screen.queryByText(TITLE)).toBeNull();
	});

	it("stays hidden after choosing don't ask again, and the choice persists", async () => {
		const user = userEvent.setup();
		const first = renderWithProviders(<PasskeyOffer />);
		await screen.findByText(TITLE);

		await user.click(
			screen.getByRole("button", { name: "*auth.passkey_offer.dont_ask*" }),
		);
		expect(screen.queryByText(TITLE)).toBeNull();
		first.unmount();

		markPasskeyOfferPending(USER_ID);
		mocks.getPasskeys.mockClear();
		renderWithProviders(<PasskeyOffer />);
		await new Promise((r) => setTimeout(r, 20));
		expect(mocks.getPasskeys).not.toHaveBeenCalled();
		expect(screen.queryByText(TITLE)).toBeNull();
	});

	it("'Not now' closes and snoozes the offer, without dismissing it for good", async () => {
		const user = userEvent.setup();
		renderWithProviders(<PasskeyOffer />);
		await screen.findByText(TITLE);

		await user.click(
			screen.getByRole("button", { name: "*auth.passkey_offer.not_now*" }),
		);

		expect(screen.queryByText(TITLE)).toBeNull();
		expect(
			localStorage.getItem(`passkey-offer:dismissed:${USER_ID}`),
		).toBeNull();
		expect(
			localStorage.getItem(`passkey-offer:snooze:${USER_ID}`),
		).not.toBeNull();
	});

	it("creates the passkey with an empty begin body and closes", async () => {
		const user = userEvent.setup();
		renderWithProviders(<PasskeyOffer />);
		await screen.findByText(TITLE);

		await user.click(screen.getByRole("button", CREATE));

		await waitFor(() =>
			expect(mocks.finishPasskeyRegistration).toHaveBeenCalledWith(
				"c1",
				expect.any(String),
				{ id: "cred" },
			),
		);
		expect(mocks.beginPasskeyRegistration).toHaveBeenCalledWith(
			undefined,
			undefined,
		);
		await waitFor(() => expect(screen.queryByText(TITLE)).toBeNull());
	});

	it("asks for the password on E-AUTH-027 and then succeeds", async () => {
		mocks.beginPasskeyRegistration.mockResolvedValueOnce(reauthError);
		const user = userEvent.setup();
		renderWithProviders(<PasskeyOffer />);
		await screen.findByText(TITLE);

		await user.click(screen.getByRole("button", CREATE));
		await user.type(
			await screen.findByLabelText("*profile.security.current_password*"),
			"password123",
		);
		expect(screen.getByText(TITLE)).toBeInTheDocument();
		expect(
			screen.queryByLabelText("*profile.two_factor.proof.code_label*"),
		).toBeNull();
		await user.click(screen.getByRole("button", CREATE));

		await waitFor(() =>
			expect(mocks.beginPasskeyRegistration).toHaveBeenLastCalledWith(
				"password123",
				undefined,
			),
		);
		await waitFor(() =>
			expect(mocks.finishPasskeyRegistration).toHaveBeenCalled(),
		);
		await waitFor(() => expect(screen.queryByText(TITLE)).toBeNull());
	});

	it("asks for the 2FA code only when 2FA is on", async () => {
		mocks.beginPasskeyRegistration.mockResolvedValueOnce(reauthError);
		mocks.getTwoFactorStatus.mockResolvedValue(
			ok({ enabled: true, recovery_codes_remaining: 8 }),
		);
		const user = userEvent.setup();
		renderWithProviders(<PasskeyOffer />);
		await screen.findByText(TITLE);

		await user.click(screen.getByRole("button", CREATE));
		await user.type(
			await screen.findByLabelText("*profile.security.current_password*"),
			"password123",
		);
		await user.type(
			await screen.findByLabelText("*profile.two_factor.proof.code_label*"),
			"123456",
		);
		await user.click(screen.getByRole("button", CREATE));

		await waitFor(() =>
			expect(mocks.beginPasskeyRegistration).toHaveBeenLastCalledWith(
				"password123",
				"123456",
			),
		);
	});

	it("a cancelled prompt is silent and keeps the dialog open", async () => {
		mocks.startRegistration.mockRejectedValue(
			Object.assign(new Error("cancelled"), { name: "NotAllowedError" }),
		);
		const user = userEvent.setup();
		renderWithProviders(<PasskeyOffer />);
		await screen.findByText(TITLE);

		await user.click(screen.getByRole("button", CREATE));

		await waitFor(() => expect(mocks.startRegistration).toHaveBeenCalled());
		expect(mocks.finishPasskeyRegistration).not.toHaveBeenCalled();
		expect(mocks.errorToast).not.toHaveBeenCalled();
		expect(screen.getByText(TITLE)).toBeInTheDocument();
	});

	it("other browser errors use the browser_error toast", async () => {
		mocks.startRegistration.mockRejectedValue(new Error("boom"));
		const user = userEvent.setup();
		renderWithProviders(<PasskeyOffer />);
		await screen.findByText(TITLE);

		await user.click(screen.getByRole("button", CREATE));

		await waitFor(() =>
			expect(mocks.errorToast).toHaveBeenCalledWith(
				null,
				"*auth.passkey.browser_error*",
			),
		);
	});

	it("works when storage throws", async () => {
		const user = userEvent.setup();
		renderWithProviders(<PasskeyOffer />);
		await screen.findByText(TITLE);
		const boom = () => {
			throw new Error("denied");
		};
		vi.spyOn(Storage.prototype, "getItem").mockImplementation(boom);
		vi.spyOn(Storage.prototype, "setItem").mockImplementation(boom);
		vi.spyOn(Storage.prototype, "removeItem").mockImplementation(boom);

		await user.click(
			screen.getByRole("button", { name: "*auth.passkey_offer.dont_ask*" }),
		);
		expect(screen.queryByText(TITLE)).toBeNull();
		vi.restoreAllMocks();
	});

	it("does not offer when storage throws on read", async () => {
		vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
			throw new Error("denied");
		});
		renderWithProviders(<PasskeyOffer />);

		await new Promise((r) => setTimeout(r, 20));
		expect(screen.queryByText(TITLE)).toBeNull();
		vi.restoreAllMocks();
	});
});
