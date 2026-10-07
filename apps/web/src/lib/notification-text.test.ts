import { useMessageStore, useText } from "@pengi/shared";
import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { getNotificationText } from "@/lib/notification-text";

describe("getNotificationText", () => {
	beforeEach(() => {
		useMessageStore.setState({
			lang: "es",
			messages: {
				stale: "{patient_name} desde hace {elapsed}",
				plain: "Sin parámetros",
				"notification.whatsapp.message": "{name}: {preview}",
				"notification.whatsapp.message.one": "{name}: {preview}",
				"notification.whatsapp.message.other":
					"{name} ({count} mensajes): {preview}",
				"whatsapp.inbox.content.image": "Imagen",
			},
		});
	});

	const text = () => renderHook(() => useText()).result.current;

	it("pluralizes WhatsApp message notifications by their numeric count", () => {
		const n = (params: Record<string, unknown>) => ({
			message_key: "notification.whatsapp.message",
			params: params as Record<string, string>,
		});
		expect(
			getNotificationText(
				n({ name: "Ana", count: 1, preview: "Hola" }),
				text(),
			),
		).toBe("Ana: Hola");
		expect(
			getNotificationText(
				n({ name: "Ana", count: 3, preview: "Hola" }),
				text(),
			),
		).toBe("Ana (3 mensajes): Hola");
		// Older notifications may carry count as a string.
		expect(
			getNotificationText(
				n({ name: "Ana", count: "2", preview: "Hola" }),
				text(),
			),
		).toBe("Ana (2 mensajes): Hola");
	});

	it("describes a WhatsApp photo without text instead of an empty preview", () => {
		expect(
			getNotificationText(
				{
					message_key: "notification.whatsapp.message",
					params: {
						name: "Ana",
						count: "1",
						preview: "",
						content_type: "image",
					},
				},
				text(),
			),
		).toBe("Ana: Imagen");
	});

	it("resolves the template and fills its params", () => {
		expect(
			getNotificationText(
				{ message_key: "plain", params: { unused: "value" } },
				text(),
			),
		).toBe("Sin parámetros");
	});

	it("computes elapsed from draft_updated_at, without the 'hace' suffix", () => {
		const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();
		expect(
			getNotificationText(
				{
					message_key: "stale",
					params: { patient_name: "Juan", draft_updated_at: twoHoursAgo },
				},
				text(),
			),
		).toBe("Juan desde hace alrededor de 2 horas");
	});

	it("derives elapsed for older notifications that only carry minutes_elapsed", () => {
		const oneHourAgo = new Date(Date.now() - 60 * 60 * 1000).toISOString();
		expect(
			getNotificationText(
				{
					message_key: "stale",
					params: { patient_name: "Juan", minutes_elapsed: "120" },
					CreatedAt: oneHourAgo,
				},
				text(),
			),
		).toBe("Juan desde hace alrededor de 3 horas");
	});
});
