import { describe, expect, it } from "vitest";
import {
	formatRelativeTime,
	getNotificationText,
	renderNotificationText,
} from "@/lib/notification-text";

describe("renderNotificationText", () => {
	it("substitutes all placeholders with matching params", () => {
		const result = renderNotificationText(
			"Tienes una consulta sin guardar para {{patient_name}} desde hace {{minutes_elapsed}} minutos.",
			{ patient_name: "Juan Perez", minutes_elapsed: "75" },
		);
		expect(result).toBe(
			"Tienes una consulta sin guardar para Juan Perez desde hace 75 minutos.",
		);
	});

	it("falls back to an empty string for missing params", () => {
		const result = renderNotificationText("Hola {{name}}!", {});
		expect(result).toBe("Hola !");
	});

	it("returns the template unchanged when it has no placeholders", () => {
		const result = renderNotificationText("Plain text", { unused: "value" });
		expect(result).toBe("Plain text");
	});
});

describe("formatRelativeTime", () => {
	it("returns empty string for a missing date", () => {
		expect(formatRelativeTime(undefined, "es")).toBe("");
		expect(formatRelativeTime(null, "es")).toBe("");
	});

	it("returns empty string for a malformed date instead of throwing", () => {
		expect(formatRelativeTime("not-a-date", "es")).toBe("");
	});

	it("formats a valid ISO date", () => {
		const oneHourAgo = new Date(Date.now() - 60 * 60 * 1000).toISOString();
		const result = formatRelativeTime(oneHourAgo, "es");
		expect(result.length).toBeGreaterThan(0);
	});
});

describe("getNotificationText", () => {
	const textGet = (key: string) =>
		key === "stale" ? "{{patient_name}} desde hace {{elapsed}}" : key;

	it("resolves the template and fills its params", () => {
		const result = getNotificationText(
			{ message_key: "stale", params: { patient_name: "Juan" } },
			textGet,
			"es",
		);
		expect(result).toBe("Juan desde hace ");
	});

	it("computes elapsed from draft_updated_at", () => {
		const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();
		const result = getNotificationText(
			{
				message_key: "stale",
				params: { patient_name: "Juan", draft_updated_at: twoHoursAgo },
			},
			textGet,
			"es",
		);
		expect(result).toBe("Juan desde hace alrededor de 2 horas");
	});
});
