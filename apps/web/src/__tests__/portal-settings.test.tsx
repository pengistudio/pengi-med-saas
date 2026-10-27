import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import PortalSettingsPage from "@/pages/settings/portal-settings";
import { renderWithProviders } from "@/test/test-wrapper";

const mockGet = vi.fn();
const mockUpdate = vi.fn();

vi.mock("@/api/portal-settings-service", async (importOriginal) => ({
	...(await importOriginal<typeof import("@/api/portal-settings-service")>()),
	getPortalSettings: () => mockGet(),
	updatePortalSettings: (s: unknown) => mockUpdate(s),
}));

vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));

const booking = {
	enabled: false,
	choose_by: "doctor",
	auto_confirm: false,
	new_patients: false,
	request_expiry_hours: 24,
	min_notice_hours: 2,
	max_days_ahead: 60,
	cancel_notice_hours: 24,
	slot_interval_minutes: 0,
};

const loaded = {
	enabled: false,
	sections: {
		appointments: true,
		results: true,
		documents: false,
		invoices: true,
	},
	booking,
	portal_url: "https://portal.test/clinic",
};

// Messages are not loaded in tests: useText renders each key as *key*.
const toggle = (key: string) =>
	screen.getByRole("switch", { name: `*${key}*` });

describe("PortalSettingsPage", () => {
	beforeEach(() => {
		mockGet.mockReset();
		mockUpdate.mockReset();
		mockGet.mockResolvedValue({ success: true, data: loaded });
		mockUpdate.mockResolvedValue({ success: true, data: loaded });
	});

	it("loads settings, enables sections with the portal and saves", async () => {
		renderWithProviders(<PortalSettingsPage />);
		expect(await screen.findByText("https://portal.test/clinic")).toBeDefined();

		const results = toggle("settings.portal.sections.results");
		expect(results.getAttribute("aria-disabled")).toBe("true");

		fireEvent.click(toggle("settings.portal.enabled"));
		expect(results.getAttribute("aria-disabled")).not.toBe("true");

		fireEvent.click(toggle("settings.portal.sections.documents"));
		fireEvent.click(screen.getByText("*settings.portal.save*"));

		await waitFor(() =>
			expect(mockUpdate).toHaveBeenCalledWith({
				enabled: true,
				sections: {
					appointments: true,
					results: true,
					documents: true,
					invoices: true,
				},
				booking,
			}),
		);
	});

	it("turns booking on only with the portal and its appointments section", async () => {
		renderWithProviders(<PortalSettingsPage />);
		await screen.findByText("https://portal.test/clinic");

		const bookingSwitch = toggle("settings.portal.booking.enabled");
		expect(bookingSwitch.getAttribute("aria-disabled")).toBe("true");
		expect(
			toggle("settings.portal.booking.new_patients").getAttribute(
				"aria-disabled",
			),
		).toBe("true");

		fireEvent.click(toggle("settings.portal.enabled"));
		expect(bookingSwitch.getAttribute("aria-disabled")).not.toBe("true");
		fireEvent.click(bookingSwitch);
		fireEvent.click(toggle("settings.portal.booking.auto_confirm"));
		fireEvent.click(toggle("settings.portal.booking.new_patients"));
		fireEvent.click(screen.getByText("*settings.portal.save*"));

		await waitFor(() =>
			expect(mockUpdate).toHaveBeenCalledWith(
				expect.objectContaining({
					enabled: true,
					booking: {
						...booking,
						enabled: true,
						auto_confirm: true,
						new_patients: true,
					},
				}),
			),
		);
	});
});
