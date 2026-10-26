import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import PortalSettingsPage from "@/pages/settings/portal-settings";

const mockGet = vi.fn();
const mockUpdate = vi.fn();

vi.mock("@/api/portal-settings-service", () => ({
	getPortalSettings: () => mockGet(),
	updatePortalSettings: (s: unknown) => mockUpdate(s),
}));

vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));

vi.mock("@pengi/shared", async (importOriginal) => ({
	...(await importOriginal<typeof import("@pengi/shared")>()),
	useText: () => ({ textGet: (key: string) => key }),
}));

const loaded = {
	enabled: false,
	sections: {
		appointments: true,
		results: true,
		documents: false,
		invoices: true,
	},
	portal_url: "https://portal.test/clinic",
};

describe("PortalSettingsPage", () => {
	beforeEach(() => {
		mockGet.mockReset();
		mockUpdate.mockReset();
		mockGet.mockResolvedValue({ success: true, data: loaded });
		mockUpdate.mockResolvedValue({ success: true, data: loaded });
	});

	it("loads settings, enables sections with the portal and saves", async () => {
		render(<PortalSettingsPage />);
		expect(await screen.findByText("https://portal.test/clinic")).toBeDefined();

		const results = screen.getByRole("switch", {
			name: "settings.portal.sections.results",
		});
		expect(results.getAttribute("aria-disabled")).toBe("true");

		fireEvent.click(
			screen.getByRole("switch", { name: "settings.portal.enabled" }),
		);
		expect(results.getAttribute("aria-disabled")).not.toBe("true");

		fireEvent.click(
			screen.getByRole("switch", {
				name: "settings.portal.sections.documents",
			}),
		);
		fireEvent.click(screen.getByText("settings.portal.save"));

		await waitFor(() =>
			expect(mockUpdate).toHaveBeenCalledWith({
				enabled: true,
				sections: {
					appointments: true,
					results: true,
					documents: true,
					invoices: true,
				},
			}),
		);
	});
});
