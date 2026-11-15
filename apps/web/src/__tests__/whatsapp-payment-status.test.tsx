import { screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WhatsAppAccount } from "@/api/whatsapp-service";
import { WhatsAppSettings } from "@/sections/settings/whatsapp-settings";
import { renderWithProviders } from "@/test/test-wrapper";

const mocks = vi.hoisted(() => ({
	account: undefined as unknown as WhatsAppAccount,
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

const HINT = "*settings.whatsapp.payment_needed.hint*";

const account = (extra: Partial<WhatsAppAccount>): WhatsAppAccount => ({
	connected: true,
	mode: "manual",
	status: "connected",
	phone_number_id: "PHONE-1",
	template_status: "APPROVED",
	reminders_enabled: true,
	reminder_offsets: [24],
	...extra,
});

describe("WhatsAppSettings — payment status", () => {
	beforeEach(() => {
		mocks.account = account({});
	});

	it("warns with a link to Meta billing when a send was rejected for payment", async () => {
		mocks.account = account({ payment_status: "needs_payment" });
		renderWithProviders(<WhatsAppSettings />);
		await waitFor(() => expect(screen.getByText(HINT)).toBeDefined());
		const link = screen
			.getByText("*settings.whatsapp.payment_needed.link*")
			.closest("a");
		expect(link?.getAttribute("href")).toContain("business.facebook.com");
		expect(link?.getAttribute("rel")).toContain("noopener");
	});

	it.each(["ok", "unknown", undefined] as const)(
		"shows no warning when payment_status is %s",
		async (payment_status) => {
			mocks.account = account({ payment_status });
			renderWithProviders(<WhatsAppSettings />);
			await waitFor(() =>
				expect(
					screen.getByText("*settings.whatsapp.status.connected*"),
				).toBeDefined(),
			);
			expect(screen.queryByText(HINT)).toBeNull();
		},
	);

	it("leaves the reconnect warning alone when the token is invalid", async () => {
		mocks.account = account({
			status: "token_invalid",
			payment_status: "needs_payment",
		});
		renderWithProviders(<WhatsAppSettings />);
		await waitFor(() =>
			expect(
				screen.getByText("*settings.whatsapp.token_invalid.hint*"),
			).toBeDefined(),
		);
		expect(screen.queryByText(HINT)).toBeNull();
	});
});
