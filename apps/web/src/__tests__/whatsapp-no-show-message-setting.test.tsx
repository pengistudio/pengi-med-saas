import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WhatsAppAccount } from "@/api/whatsapp-service";
import { WhatsAppSettings } from "@/sections/settings/whatsapp-settings";
import { renderWithProviders } from "@/test/test-wrapper";

const mocks = vi.hoisted(() => ({
	account: undefined as unknown as WhatsAppAccount,
	update: vi.fn(),
}));

vi.mock("@/api/whatsapp-service", async (importOriginal) => {
	const actual =
		await importOriginal<typeof import("@/api/whatsapp-service")>();
	return {
		...actual,
		getWhatsAppConfig: vi.fn(async () => ({
			success: true,
			data: { embedded_signup_available: false },
		})),
		getWhatsAppAccount: vi.fn(async () => ({
			success: true,
			data: mocks.account,
		})),
		updateWhatsAppSettings: mocks.update,
	};
});
vi.mock("@/hooks/use-permission", () => ({
	default: () => ({ checkPermission: () => false }),
}));
vi.mock("@/hooks/use-embedded-signup", () => ({
	useEmbeddedSignup: () => ({ running: false, start: vi.fn() }),
}));
vi.mock("@/sections/settings/whatsapp-templates-usage", () => ({
	StatusPill: () => null,
	WhatsAppTemplateList: () => null,
	WhatsAppUsageBar: () => null,
}));

// Messages are not loaded in tests: useText renders each key as *key*.
const toggle = () =>
	screen.getByRole("switch", { name: "*settings.whatsapp.no_show_message*" });

describe("WhatsAppSettings — no-show message", () => {
	beforeEach(() => {
		mocks.account = {
			connected: true,
			mode: "manual",
			status: "connected",
			phone_number_id: "PHONE-1",
			template_status: "APPROVED",
			reminders_enabled: true,
			reminder_offsets: [24],
		};
		mocks.update.mockReset();
		mocks.update.mockImplementation(async (payload) => ({
			success: true,
			data: { ...mocks.account, ...payload },
		}));
	});

	it("is off by default and saves the switch with the reminder settings", async () => {
		renderWithProviders(<WhatsAppSettings />);
		await waitFor(() => expect(toggle()).toBeDefined());
		expect(toggle().getAttribute("aria-checked")).toBe("false");
		expect(
			screen.getByText("*settings.whatsapp.no_show_message.hint*"),
		).toBeDefined();
		const save = screen.getByText("*form.save*").closest("button");
		expect(save?.disabled).toBe(true);

		fireEvent.click(toggle());
		expect(save?.disabled).toBe(false);
		fireEvent.click(save as HTMLButtonElement);
		await waitFor(() =>
			expect(mocks.update).toHaveBeenCalledWith({
				reminders_enabled: true,
				reminder_offsets: [24],
				no_show_message_enabled: true,
			}),
		);
		await waitFor(() => expect(save?.disabled).toBe(true));
		expect(toggle().getAttribute("aria-checked")).toBe("true");
	});

	it("shows the stored value", async () => {
		mocks.account = { ...mocks.account, no_show_message_enabled: true };
		renderWithProviders(<WhatsAppSettings />);
		await waitFor(() =>
			expect(toggle().getAttribute("aria-checked")).toBe("true"),
		);
	});
});
