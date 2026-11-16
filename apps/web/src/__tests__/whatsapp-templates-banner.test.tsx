import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { WhatsAppAccount } from "@/api/whatsapp-service";
import {
	templatesState,
	WhatsAppTemplatesBanner,
} from "@/sections/settings/whatsapp-templates-banner";
import { renderWithProviders } from "@/test/test-wrapper";

const base: WhatsAppAccount = {
	connected: true,
	status: "connected",
	template_status: "",
	reminders_enabled: false,
	reminder_offsets: [],
	missing_templates: [],
	templates: [],
};

function renderBanner(account: WhatsAppAccount) {
	const onCreate = vi.fn();
	const onSync = vi.fn();
	const view = renderWithProviders(
		<WhatsAppTemplatesBanner
			account={account}
			busy={null}
			onCreate={onCreate}
			onSync={onSync}
		/>,
	);
	return { ...view, onCreate, onSync };
}

// Messages are not loaded in tests: useText renders each key as *key*.
describe("WhatsAppTemplatesBanner", () => {
	it("missing: offers to create them, only on click", () => {
		const { onCreate, onSync } = renderBanner({
			...base,
			templates_state: "missing",
			missing_templates: ["portal_code"],
		});
		expect(
			screen.getByText("*settings.whatsapp.templates_state.missing.title*"),
		).toBeDefined();
		expect(onCreate).not.toHaveBeenCalled();
		fireEvent.click(
			screen.getByRole("button", {
				name: "*settings.whatsapp.template.create_missing*",
			}),
		);
		expect(onCreate).toHaveBeenCalledTimes(1);
		expect(onSync).not.toHaveBeenCalled();
	});

	it("pending: says Meta is reviewing and offers a refresh", () => {
		const { onCreate, onSync } = renderBanner({
			...base,
			templates_state: "pending",
		});
		expect(
			screen.getByText(
				"*settings.whatsapp.templates_state.pending.description*",
			),
		).toBeDefined();
		expect(
			screen.queryByRole("button", {
				name: "*settings.whatsapp.template.create_missing*",
			}),
		).toBeNull();
		fireEvent.click(
			screen.getByRole("button", {
				name: "*settings.whatsapp.templates_state.refresh*",
			}),
		);
		expect(onSync).toHaveBeenCalledTimes(1);
		expect(onCreate).not.toHaveBeenCalled();
	});

	it("rejected: lists the refused templates with Meta's reason", () => {
		renderBanner({
			...base,
			templates_state: "rejected",
			templates: [
				{ name: "pengi_cita_recordatorio", status: "APPROVED" },
				{
					name: "pengi_cita_confirmada",
					status: "REJECTED",
					reason: "INVALID_FORMAT",
				},
				{ name: "pengi_cita_no_asistio", status: "PAUSED" },
			],
		});
		expect(
			screen.getByText("*settings.whatsapp.templates_state.rejected.title*"),
		).toBeDefined();
		expect(screen.getByText("pengi_cita_confirmada")).toBeDefined();
		expect(screen.getByText("INVALID_FORMAT")).toBeDefined();
		expect(screen.getByText("pengi_cita_no_asistio")).toBeDefined();
		expect(screen.queryByText("pengi_cita_recordatorio")).toBeNull();
	});

	it("rejected: ignores optional templates", () => {
		renderBanner({
			...base,
			templates_state: "rejected",
			templates: [
				{ name: "portal_code", status: "REJECTED", optional: true },
				{ name: "pengi_cita_no_asistio", status: "PAUSED" },
			],
		});
		expect(screen.queryByText("portal_code")).toBeNull();
		expect(screen.getByText("pengi_cita_no_asistio")).toBeDefined();
	});

	it("ready or not connected: nothing", () => {
		renderBanner({ ...base, templates_state: "ready" });
		expect(
			screen.queryByText(/settings\.whatsapp\.templates_state/),
		).toBeNull();
		expect(templatesState({ ...base, connected: false })).toBe("");
		// Older API without templates_state: derived from the missing list.
		expect(
			templatesState({ ...base, missing_templates: ["portal_code"] }),
		).toBe("missing");
	});
});
