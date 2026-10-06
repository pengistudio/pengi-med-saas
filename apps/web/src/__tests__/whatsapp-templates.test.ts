import { describe, expect, it } from "vitest";
import type { WhatsAppTemplate } from "@/api/whatsapp-service";
import {
	inboxTemplates,
	templateNeedsAppointment,
	templatePreview,
	usageSummary,
} from "@/lib/whatsapp-templates";

function template(overrides: Partial<WhatsAppTemplate>): WhatsAppTemplate {
	return {
		name: "pengi_continuar_conversacion",
		status: "APPROVED",
		reason: "",
		variables: ["patient", "clinic"],
		needs_appointment: false,
		inbox: true,
		usable: true,
		body: "Hola {{1}}",
		preview: "Hola Ana",
		buttons: [],
		...overrides,
	};
}

describe("usageSummary", () => {
	it("treats -1 as unlimited", () => {
		expect(usageSummary({ used: 500, limit: -1 })).toEqual({
			unlimited: true,
			percent: 0,
			warning: false,
			reached: false,
		});
	});

	it("computes the rounded percentage below the warning share", () => {
		expect(usageSummary({ used: 120, limit: 300 })).toEqual({
			unlimited: false,
			percent: 40,
			warning: false,
			reached: false,
		});
	});

	it("warns from 80% until the cap", () => {
		expect(usageSummary({ used: 240, limit: 300 }).warning).toBe(true);
		expect(usageSummary({ used: 239, limit: 300 }).warning).toBe(false);
	});

	it("is reached at the cap and caps the bar at 100% when over it", () => {
		expect(usageSummary({ used: 300, limit: 300 })).toMatchObject({
			percent: 100,
			reached: true,
			warning: false,
		});
		// Concurrent sends can go slightly over.
		expect(usageSummary({ used: 305, limit: 300 }).percent).toBe(100);
	});

	it("is reached right away with a zero cap", () => {
		expect(usageSummary({ used: 0, limit: 0 }).reached).toBe(true);
	});
});

describe("templateNeedsAppointment", () => {
	it("follows needs_appointment", () => {
		expect(
			templateNeedsAppointment(template({ needs_appointment: true })),
		).toBe(true);
		expect(templateNeedsAppointment(template({}))).toBe(false);
	});

	it("also asks for one when the template fills a date or time", () => {
		expect(
			templateNeedsAppointment(
				template({ variables: ["patient", "clinic", "date", "time"] }),
			),
		).toBe(true);
	});
});

describe("templatePreview", () => {
	it("prefers the sample-filled preview and falls back to the body", () => {
		expect(templatePreview(template({}))).toBe("Hola Ana");
		expect(templatePreview(template({ preview: "" }))).toBe("Hola {{1}}");
	});
});

describe("inboxTemplates", () => {
	it("drops the reminder and lists usable templates first", () => {
		const list = inboxTemplates([
			template({
				name: "pengi_cita_recordatorio",
				inbox: false,
				usable: false,
			}),
			template({ name: "pending", status: "PENDING", usable: false }),
			template({ name: "approved" }),
		]);
		expect(list.map((t) => t.name)).toEqual(["approved", "pending"]);
	});
});
