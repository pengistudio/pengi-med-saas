import { describe, expect, it } from "vitest";
import {
	MAX_REMINDER_OFFSETS,
	messageErrorKey,
	toggleReminderOffset,
} from "@/api/whatsapp-service";
import { parseEmbeddedSignupMessage } from "@/lib/facebook-sdk";

describe("toggleReminderOffset", () => {
	it("adds an offset keeping them from earliest to latest reminder", () => {
		expect(toggleReminderOffset([2], 24)).toEqual([24, 2]);
		expect(toggleReminderOffset([48, 2], 12)).toEqual([48, 12, 2]);
	});

	it("removes a selected offset", () => {
		expect(toggleReminderOffset([24, 2], 24)).toEqual([2]);
	});

	it("ignores additions past the limit", () => {
		const full = [48, 24, 12];
		expect(full).toHaveLength(MAX_REMINDER_OFFSETS);
		expect(toggleReminderOffset(full, 1)).toEqual(full);
		// Removing still works when full.
		expect(toggleReminderOffset(full, 24)).toEqual([48, 12]);
	});
});

describe("messageErrorKey", () => {
	it("maps known codes to their own key", () => {
		expect(messageErrorKey("131026")).toBe("whatsapp.message.error.131026");
		expect(messageErrorKey("no_phone")).toBe("whatsapp.message.error.no_phone");
	});

	it("falls back to the generic key for unknown codes", () => {
		expect(messageErrorKey("999999")).toBe("whatsapp.message.error.unknown");
	});

	it("returns null without an error", () => {
		expect(messageErrorKey("")).toBeNull();
	});
});

describe("parseEmbeddedSignupMessage", () => {
	const finish = {
		type: "WA_EMBEDDED_SIGNUP",
		event: "FINISH",
		data: { phone_number_id: "111", waba_id: "222" },
	};

	it("reads the ids from a FINISH message (object or JSON string)", () => {
		const expected = {
			type: "finish",
			result: { waba_id: "222", phone_number_id: "111" },
		};
		expect(
			parseEmbeddedSignupMessage("https://www.facebook.com", finish),
		).toEqual(expected);
		expect(
			parseEmbeddedSignupMessage(
				"https://web.facebook.com",
				JSON.stringify(finish),
			),
		).toEqual(expected);
	});

	it("reports a cancelled flow", () => {
		expect(
			parseEmbeddedSignupMessage("https://www.facebook.com", {
				type: "WA_EMBEDDED_SIGNUP",
				event: "CANCEL",
			}),
		).toEqual({ type: "cancel" });
	});

	it("ignores other origins and other messages", () => {
		expect(
			parseEmbeddedSignupMessage("https://evil-facebook.com", finish),
		).toBeNull();
		expect(
			parseEmbeddedSignupMessage("https://facebook.com.evil.io", finish),
		).toBeNull();
		expect(
			parseEmbeddedSignupMessage("https://www.facebook.com", "not json"),
		).toBeNull();
		expect(
			parseEmbeddedSignupMessage("https://www.facebook.com", { type: "x" }),
		).toBeNull();
	});
});
