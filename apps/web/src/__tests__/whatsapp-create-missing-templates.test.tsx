import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WhatsAppAccount } from "@/api/whatsapp-service";
import { WhatsAppSettings } from "@/sections/settings/whatsapp-settings";
import { renderWithProviders } from "@/test/test-wrapper";

const mocks = vi.hoisted(() => ({
	account: undefined as unknown as WhatsAppAccount,
	createMissing: vi.fn(),
	sync: vi.fn(),
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
		createMissingTemplates: mocks.createMissing,
		syncWhatsAppTemplate: mocks.sync,
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
const createButton = () =>
	screen.queryByRole("button", {
		name: "*settings.whatsapp.template.create_missing*",
	});

describe("WhatsAppSettings — create missing templates", () => {
	beforeEach(() => {
		mocks.account = {
			connected: true,
			mode: "embedded",
			coexistence: true,
			status: "connected",
			phone_number_id: "PHONE-1",
			template_status: "",
			reminders_enabled: true,
			reminder_offsets: [24],
			missing_templates: ["pengi_cita_recordatorio", "portal_code"],
		};
		mocks.createMissing.mockReset();
		mocks.createMissing.mockImplementation(async () => ({
			success: true,
			data: {
				...mocks.account,
				template_status: "PENDING",
				missing_templates: [],
			},
		}));
		mocks.sync.mockReset();
	});

	it("is shown when a template is missing and is never called on its own", async () => {
		renderWithProviders(<WhatsAppSettings />);
		await waitFor(() => expect(createButton()).not.toBeNull());
		expect(
			screen.getByText("*settings.whatsapp.template.missing_count*"),
		).toBeDefined();
		expect(mocks.createMissing).not.toHaveBeenCalled();
	});

	it("creates them on click and hides once none is missing", async () => {
		renderWithProviders(<WhatsAppSettings />);
		await waitFor(() => expect(createButton()).not.toBeNull());
		fireEvent.click(createButton() as HTMLButtonElement);
		await waitFor(() => expect(mocks.createMissing).toHaveBeenCalledTimes(1));
		await waitFor(() => expect(createButton()).toBeNull());
		expect(mocks.sync).not.toHaveBeenCalled();
	});

	it("is hidden when every template exists", async () => {
		mocks.account = { ...mocks.account, missing_templates: [] };
		renderWithProviders(<WhatsAppSettings />);
		await waitFor(() =>
			expect(
				screen.getByText("*settings.whatsapp.template.sync*"),
			).toBeDefined(),
		);
		expect(createButton()).toBeNull();
	});
});
