import type { AppText } from "@pengi/shared";
import { describe, expect, it } from "vitest";
import { SUPPORT } from "@/lib/constants";
import { supportLinks } from "@/lib/support-links";

// Fills {placeholders} like useText does, from a tiny catalog.
const catalog: Record<string, string> = {
	"dashboard.help.whatsapp.message": "Hola, soy de {clinic} ({slug}): {page}",
	"dashboard.help.report.message": "Problema en {page}",
	"dashboard.help.email.subject": "Soporte: {clinic}",
	"dashboard.help.email.body": "Clínica: {clinic}\nPantalla: {page}",
};
const textGet = ((key: string, params: Record<string, string> = {}) =>
	(catalog[key] ?? key).replace(
		/\{(\w+)\}/g,
		(_, name: string) => params[name] ?? "",
	)) as AppText["textGet"];

const context = {
	clinic: "Clínica Ñandú & Co",
	slug: "nandu",
	page: "https://app.example.com/billing?x=1",
};

describe("supportLinks", () => {
	const links = supportLinks(context, textGet);

	it("writes the clinic and the page in the WhatsApp message", () => {
		const url = new URL(links.whatsapp);
		expect(url.origin + url.pathname).toBe(`https://wa.me/${SUPPORT.WHATSAPP}`);
		expect(url.searchParams.get("text")).toBe(
			"Hola, soy de Clínica Ñandú & Co (nandu): https://app.example.com/billing?x=1",
		);
	});

	it("reports a problem over WhatsApp with its own message", () => {
		const url = new URL(links.report);
		expect(url.searchParams.get("text")).toBe(
			"Problema en https://app.example.com/billing?x=1",
		);
	});

	it("fills the email subject and body, encoded", () => {
		expect(links.email.startsWith(`mailto:${SUPPORT.EMAIL}?`)).toBe(true);
		const params = new URLSearchParams(links.email.split("?")[1]);
		expect(params.get("subject")).toBe("Soporte: Clínica Ñandú & Co");
		expect(params.get("body")).toBe(
			"Clínica: Clínica Ñandú & Co\nPantalla: https://app.example.com/billing?x=1",
		);
	});
});
