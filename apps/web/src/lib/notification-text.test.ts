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
			},
		});
	});

	const text = () => renderHook(() => useText()).result.current;

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
