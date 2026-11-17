import { fireEvent, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PERMISSIONS } from "@/lib/constants";
import { WhatsAppSettingsLink } from "@/sections/whatsapp/whatsapp-settings-link";
import { renderWithProviders } from "@/test/test-wrapper";

let granted: string[] = [];
vi.mock("@/hooks/use-permission", () => ({
	default: () => ({
		checkPermission: (p: string[]) => p.every((x) => granted.includes(x)),
		permissions: granted,
	}),
}));

const renderLink = () =>
	renderWithProviders(
		<MemoryRouter>
			<Routes>
				<Route path="/" element={<WhatsAppSettingsLink />} />
				<Route path="/settings/whatsapp" element={<p>settings page</p>} />
			</Routes>
		</MemoryRouter>,
	);

describe("WhatsAppSettingsLink", () => {
	beforeEach(() => {
		granted = [];
	});

	it("navigates to /settings/whatsapp with the manage permission", () => {
		granted = [PERMISSIONS.WHATSAPP.PERMISSION_MANAGE_WHATSAPP];
		renderLink();
		fireEvent.click(
			screen.getByRole("button", { name: "*whatsapp.inbox.settings*" }),
		);
		expect(screen.getByText("settings page")).toBeDefined();
	});

	it("is absent without it", () => {
		granted = [PERMISSIONS.WHATSAPP.PERMISSION_USE_WHATSAPP_INBOX];
		renderLink();
		expect(screen.queryByRole("button")).toBeNull();
	});
});
