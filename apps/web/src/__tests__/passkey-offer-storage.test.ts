import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
	clearLocalStorageKeepingOffer,
	dismissPasskeyOffer,
	isPasskeyOfferDismissed,
	snoozePasskeyOffer,
} from "@/lib/passkey-offer-storage";

const USER = 7;
const DAY = 24 * 60 * 60 * 1000;

describe("passkey offer snooze", () => {
	beforeEach(() => {
		localStorage.clear();
		vi.useFakeTimers();
		vi.setSystemTime(new Date("2026-11-15T12:00:00Z"));
	});
	afterEach(() => {
		vi.useRealTimers();
		localStorage.clear();
	});

	it("is shown until the user says not now", () => {
		expect(isPasskeyOfferDismissed(USER)).toBe(false);
	});

	it("'not now' silences it for 30 days, then asks again", () => {
		snoozePasskeyOffer(USER);
		expect(isPasskeyOfferDismissed(USER)).toBe(true);

		vi.advanceTimersByTime(29 * DAY);
		expect(isPasskeyOfferDismissed(USER)).toBe(true);

		vi.advanceTimersByTime(2 * DAY);
		expect(isPasskeyOfferDismissed(USER)).toBe(false);
	});

	it("the third 'not now' stops it for good", () => {
		for (let i = 0; i < 3; i++) {
			snoozePasskeyOffer(USER);
			vi.advanceTimersByTime(31 * DAY);
		}
		expect(isPasskeyOfferDismissed(USER)).toBe(true);
		expect(localStorage.getItem(`passkey-offer:dismissed:${USER}`)).toBe("1");
	});

	it("is per user", () => {
		snoozePasskeyOffer(USER);
		expect(isPasskeyOfferDismissed(USER + 1)).toBe(false);
	});

	it("survives sign-out together with 'don't ask again'", () => {
		snoozePasskeyOffer(USER);
		dismissPasskeyOffer(USER + 1);
		localStorage.setItem("token", "secret");

		clearLocalStorageKeepingOffer();

		expect(localStorage.getItem("token")).toBeNull();
		expect(isPasskeyOfferDismissed(USER)).toBe(true);
		expect(isPasskeyOfferDismissed(USER + 1)).toBe(true);
	});

	it("ignores a corrupt snooze entry", () => {
		localStorage.setItem(`passkey-offer:snooze:${USER}`, "{not json");
		expect(isPasskeyOfferDismissed(USER)).toBe(false);
	});
});
