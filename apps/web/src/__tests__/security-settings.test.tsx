import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useSessionStore } from "@/store/session-store";
import { renderWithProviders } from "@/test/test-wrapper";

const mocks = vi.hoisted(() => ({ updateSecuritySettings: vi.fn() }));
vi.mock("@/api/settings-service", () => ({
	updateSecuritySettings: mocks.updateSecuritySettings,
}));

import { SecuritySettings } from "@/sections/settings/security-settings";

const ENVIRONMENT = {
	id: 1,
	name: "Admin",
	role: "admin",
	role_id: 1,
	company_id: 1,
	legal_name: "Clinic",
	trade_name: "Clinic",
	plan_code: "pro",
	tenant_id: 1,
	tenant_name: "Clinic",
	tenant_slug: "clinic",
	permissions: [],
};

const ok = <T,>(data: T) => ({ success: true, code: 200, message: "", data });

describe("SecuritySettings", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		useSessionStore.setState({
			environment: { ...ENVIRONMENT, require_mfa: false },
		});
	});

	it("turns the requirement on and keeps it in the session", async () => {
		const user = userEvent.setup();
		mocks.updateSecuritySettings.mockResolvedValue(ok({ require_mfa: true }));
		renderWithProviders(<SecuritySettings />);

		const toggle = screen.getByRole("switch");
		expect(toggle).toHaveAttribute("aria-checked", "false");
		expect(
			screen.getByText("*settings.security.require_mfa.hint*"),
		).toBeInTheDocument();

		await user.click(toggle);

		expect(mocks.updateSecuritySettings).toHaveBeenCalledWith({
			require_mfa: true,
		});
		await waitFor(() =>
			expect(screen.getByRole("switch")).toHaveAttribute(
				"aria-checked",
				"true",
			),
		);
		expect(useSessionStore.getState().environment?.require_mfa).toBe(true);
	});

	it("stays off when the API refuses (admin without a second factor)", async () => {
		const user = userEvent.setup();
		mocks.updateSecuritySettings.mockResolvedValue({
			success: false,
			code: 409,
			message: "",
			data: { error_code: "E-TEN-006", error_message: "" },
		});
		renderWithProviders(<SecuritySettings />);

		await user.click(screen.getByRole("switch"));

		await waitFor(() => expect(screen.getByRole("switch")).not.toBeDisabled());
		expect(screen.getByRole("switch")).toHaveAttribute("aria-checked", "false");
		expect(useSessionStore.getState().environment?.require_mfa).toBe(false);
	});

	it("turns an active requirement off", async () => {
		const user = userEvent.setup();
		useSessionStore.setState({
			environment: { ...ENVIRONMENT, require_mfa: true },
		});
		mocks.updateSecuritySettings.mockResolvedValue(ok({ require_mfa: false }));
		renderWithProviders(<SecuritySettings />);

		await user.click(screen.getByRole("switch"));

		expect(mocks.updateSecuritySettings).toHaveBeenCalledWith({
			require_mfa: false,
		});
		await waitFor(() =>
			expect(useSessionStore.getState().environment?.require_mfa).toBe(false),
		);
	});
});
