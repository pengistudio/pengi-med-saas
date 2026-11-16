import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WhatsAppAccount } from "@/api/whatsapp-service";
import { WhatsAppSettings } from "@/sections/settings/whatsapp-settings";
import { renderWithProviders } from "@/test/test-wrapper";

// Connecting a real number twice got a clinic's account reviewed by Meta:
// every connect option and the disconnect ask for confirmation first.

const mocks = vi.hoisted(() => ({
	account: undefined as unknown as WhatsAppAccount,
	start: vi.fn(),
	disconnect: vi.fn(async () => ({ success: true })),
}));

vi.mock("@/api/whatsapp-service", async (importOriginal) => {
	const actual =
		await importOriginal<typeof import("@/api/whatsapp-service")>();
	return {
		...actual,
		getWhatsAppConfig: vi.fn(async () => ({
			success: true,
			data: { embedded_signup_available: true },
		})),
		getWhatsAppAccount: vi.fn(async () => ({
			success: true,
			data: mocks.account,
		})),
		disconnectWhatsApp: mocks.disconnect,
	};
});
vi.mock("@/hooks/use-permission", () => ({
	default: () => ({ checkPermission: () => false }),
}));
vi.mock("@/hooks/use-embedded-signup", () => ({
	useEmbeddedSignup: () => ({ running: false, start: mocks.start }),
}));
vi.mock("@/sections/settings/whatsapp-templates-usage", () => ({
	StatusPill: () => null,
	WhatsAppTemplateList: () => null,
	WhatsAppUsageBar: () => null,
}));
vi.mock("@/sections/settings/whatsapp-bot-settings", () => ({
	WhatsAppBotSettings: () => null,
}));

const notConnected: WhatsAppAccount = {
	connected: false,
	template_status: "",
	reminders_enabled: false,
	reminder_offsets: [],
};

async function openConfirm(button: string, title: string) {
	renderWithProviders(<WhatsAppSettings />);
	fireEvent.click(await screen.findByText(button));
	await screen.findByText(title);
}

describe("WhatsAppSettings — connect confirmation", () => {
	beforeEach(() => {
		mocks.account = notConnected;
		mocks.start.mockClear();
		mocks.disconnect.mockClear();
	});

	it.each([
		["app", "*settings.whatsapp.connect_business_app*"],
		["api", "*settings.whatsapp.connect*"],
	])(
		"cancel on the %s warning does not start the flow",
		async (kind, button) => {
			await openConfirm(button, `*settings.whatsapp.confirm.${kind}.title*`);
			fireEvent.click(screen.getByText("*form.cancel*"));
			await waitFor(() =>
				expect(
					screen.queryByText(`*settings.whatsapp.confirm.${kind}.title*`),
				).toBeNull(),
			);
			expect(mocks.start).not.toHaveBeenCalled();
		},
	);

	it("confirming the app warning starts coexistence", async () => {
		await openConfirm(
			"*settings.whatsapp.connect_business_app*",
			"*settings.whatsapp.confirm.app.title*",
		);
		expect(
			screen.getByText("*settings.whatsapp.confirm.app.description*"),
		).toBeDefined();
		fireEvent.click(screen.getByText("*form.continue*"));
		expect(mocks.start).toHaveBeenCalledTimes(1);
		expect(mocks.start).toHaveBeenCalledWith({ coexistence: true });
	});

	it("confirming the dedicated-number warning starts the API flow", async () => {
		await openConfirm(
			"*settings.whatsapp.connect*",
			"*settings.whatsapp.confirm.api.title*",
		);
		fireEvent.click(screen.getByText("*form.continue*"));
		expect(mocks.start).toHaveBeenCalledTimes(1);
		expect(mocks.start).toHaveBeenCalledWith(undefined);
	});
});

describe("WhatsAppSettings — disconnect confirmation", () => {
	beforeEach(() => {
		mocks.account = {
			connected: true,
			mode: "embedded",
			coexistence: true,
			status: "connected",
			phone_number_id: "PHONE-1",
			template_status: "APPROVED",
			reminders_enabled: true,
			reminder_offsets: [24],
		};
		mocks.disconnect.mockClear();
	});

	it("cancel keeps the connection; confirm disconnects", async () => {
		await openConfirm(
			"*settings.whatsapp.disconnect*",
			"*settings.whatsapp.disconnect.title*",
		);
		expect(
			screen.getByText("*settings.whatsapp.disconnect.description*"),
		).toBeDefined();
		fireEvent.click(screen.getByText("*form.cancel*"));
		await waitFor(() =>
			expect(
				screen.queryByText("*settings.whatsapp.disconnect.title*"),
			).toBeNull(),
		);
		expect(mocks.disconnect).not.toHaveBeenCalled();

		fireEvent.click(screen.getByText("*settings.whatsapp.disconnect*"));
		await screen.findByText("*settings.whatsapp.disconnect.title*");
		fireEvent.click(screen.getByText("*form.continue*"));
		await waitFor(() => expect(mocks.disconnect).toHaveBeenCalledTimes(1));
	});
});
