import { describe, expect, it } from "vitest";
import { replyWindow } from "@/lib/whatsapp-window";

const NOW = new Date("2026-10-05T12:00:00Z");

describe("replyWindow", () => {
	it("is closed when the patient never wrote", () => {
		expect(replyWindow(null, NOW)).toEqual({
			open: false,
			hoursLeft: 0,
			minutesLeft: 0,
		});
		expect(replyWindow(undefined, NOW).open).toBe(false);
		expect(replyWindow("not a date", NOW).open).toBe(false);
	});

	it("is closed once the expiry has passed", () => {
		expect(replyWindow("2026-10-05T11:59:59Z", NOW).open).toBe(false);
		expect(replyWindow("2026-10-05T12:00:00Z", NOW).open).toBe(false);
	});

	it("counts whole hours left while open", () => {
		expect(replyWindow("2026-10-06T11:30:00Z", NOW)).toEqual({
			open: true,
			hoursLeft: 23,
			minutesLeft: 23 * 60 + 30,
		});
	});

	it("reports minutes in the last hour, never zero while open", () => {
		expect(replyWindow("2026-10-05T12:45:00Z", NOW)).toMatchObject({
			open: true,
			hoursLeft: 0,
			minutesLeft: 45,
		});
		expect(replyWindow("2026-10-05T12:00:20Z", NOW)).toMatchObject({
			open: true,
			minutesLeft: 1,
		});
	});
});
