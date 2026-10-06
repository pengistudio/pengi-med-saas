import { afterEach, describe, expect, it, vi } from "vitest";
import type { Notification } from "@/api/notification-service";
import {
	isNotificationSoundMuted,
	setNotificationSoundMuted,
} from "@/lib/notification-sound";
import {
	detectNewWhatsAppMessages,
	isOnWhatsAppInbox,
	notificationTextParams,
} from "@/lib/whatsapp-alerts";

function notification(overrides: Partial<Notification>): Notification {
	return {
		ID: 1,
		CreatedAt: "2026-10-05T10:00:00Z",
		UpdatedAt: "2026-10-05T10:00:00Z",
		tenant_id: 1,
		user_id: 1,
		type: "whatsapp.message",
		resource_type: "whatsapp_conversation",
		resource_id: 7,
		message_key: "notification.whatsapp.message",
		params: {},
		action_url: "/whatsapp?c=7",
		level: "info",
		read_at: null,
		...overrides,
	};
}

describe("detectNewWhatsAppMessages", () => {
	it("reports nothing on the first poll, only records what is there", () => {
		const { fresh, seen } = detectNewWhatsAppMessages(null, [notification({})]);
		expect(fresh).toEqual([]);
		expect(seen.get(1)).toBe("2026-10-05T10:00:00Z");
	});

	it("reports an unseen WhatsApp message", () => {
		const first = detectNewWhatsAppMessages(null, [notification({})]);
		const next = notification({ ID: 2 });
		const { fresh } = detectNewWhatsAppMessages(first.seen, [
			next,
			notification({}),
		]);
		expect(fresh).toEqual([next]);
	});

	it("reports a seen notification again when it was updated (another message)", () => {
		const first = detectNewWhatsAppMessages(null, [notification({})]);
		const updated = notification({ UpdatedAt: "2026-10-05T10:05:00Z" });
		expect(detectNewWhatsAppMessages(first.seen, [updated]).fresh).toEqual([
			updated,
		]);
	});

	it("ignores unchanged ones and other notification types", () => {
		const first = detectNewWhatsAppMessages(null, [notification({})]);
		const other = notification({ ID: 3, type: "exam_order.result" });
		expect(
			detectNewWhatsAppMessages(first.seen, [notification({}), other]).fresh,
		).toEqual([]);
	});

	it("remembers notifications that left the page", () => {
		const first = detectNewWhatsAppMessages(null, [notification({})]);
		const second = detectNewWhatsAppMessages(first.seen, []);
		expect(
			detectNewWhatsAppMessages(second.seen, [notification({})]).fresh,
		).toEqual([]);
	});
});

describe("isOnWhatsAppInbox", () => {
	it("matches the inbox path only", () => {
		expect(isOnWhatsAppInbox("/whatsapp")).toBe(true);
		expect(isOnWhatsAppInbox("/whatsapp/x")).toBe(true);
		expect(isOnWhatsAppInbox("/whatsapp-settings")).toBe(false);
		expect(isOnWhatsAppInbox("/settings")).toBe(false);
	});
});

describe("notificationTextParams", () => {
	it("turns a numeric count into a number and keeps the rest", () => {
		expect(notificationTextParams({ name: "Ana", count: "3" })).toEqual({
			name: "Ana",
			count: 3,
		});
		expect(notificationTextParams({ count: 2, id: 7 })).toEqual({
			count: 2,
			id: 7,
		});
		expect(notificationTextParams({ count: "" })).toEqual({ count: "" });
		expect(notificationTextParams(null)).toEqual({});
	});
});

describe("notification sound preference", () => {
	afterEach(() => {
		vi.restoreAllMocks();
		localStorage.clear();
	});

	it("persists the mute in localStorage", () => {
		expect(isNotificationSoundMuted()).toBe(false);
		setNotificationSoundMuted(true);
		expect(isNotificationSoundMuted()).toBe(true);
		setNotificationSoundMuted(false);
		expect(isNotificationSoundMuted()).toBe(false);
	});

	it("survives blocked storage", () => {
		vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
			throw new Error("blocked");
		});
		vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
			throw new Error("blocked");
		});
		expect(() => setNotificationSoundMuted(true)).not.toThrow();
		expect(isNotificationSoundMuted()).toBe(false);
	});
});
