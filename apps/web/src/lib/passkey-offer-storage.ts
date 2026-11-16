/**
 * State of the "save a passkey" offer shown after a password login. Every
 * access is guarded: the app must work with storage unavailable (private
 * mode, blocked cookies), where the offer simply behaves as "ask every time".
 */
const PENDING_KEY = "passkey-offer:pending";
const DISMISSED_PREFIX = "passkey-offer:dismissed:";
const SNOOZE_PREFIX = "passkey-offer:snooze:";

/**
 * "Not now" is not a no: it silences the offer for SNOOZE_DAYS, and after
 * MAX_SNOOZES of them the offer stops for good. We don't nag.
 */
const SNOOZE_DAYS = 30;
const MAX_SNOOZES = 3;

/** Set at a password login (survives the redirects to the first screen). */
export function markPasskeyOfferPending(userId: number): void {
	try {
		sessionStorage.setItem(PENDING_KEY, String(userId));
	} catch {
		// Storage unavailable: no offer.
	}
}

/** Reads and clears the mark, so a reload does not ask again. */
export function takePasskeyOfferPending(): number | null {
	try {
		const raw = sessionStorage.getItem(PENDING_KEY);
		if (raw === null) return null;
		sessionStorage.removeItem(PENDING_KEY);
		const id = Number(raw);
		return Number.isFinite(id) ? id : null;
	} catch {
		return null;
	}
}

function readSnooze(userId: number): { until: number; count: number } {
	try {
		const raw = localStorage.getItem(SNOOZE_PREFIX + userId);
		const parsed = raw ? JSON.parse(raw) : null;
		const until = Number(parsed?.until);
		const count = Number(parsed?.count);
		return {
			until: Number.isFinite(until) ? until : 0,
			count: Number.isFinite(count) ? count : 0,
		};
	} catch {
		return { until: 0, count: 0 };
	}
}

/** True when the offer must not be shown: dismissed for good or snoozed. */
export function isPasskeyOfferDismissed(userId: number): boolean {
	try {
		if (localStorage.getItem(DISMISSED_PREFIX + userId) === "1") return true;
	} catch {
		return false;
	}
	return readSnooze(userId).until > Date.now();
}

/** "Not now": quiet for SNOOZE_DAYS; the MAX_SNOOZES-th one is final. */
export function snoozePasskeyOffer(userId: number): void {
	const count = readSnooze(userId).count + 1;
	if (count >= MAX_SNOOZES) {
		dismissPasskeyOffer(userId);
		return;
	}
	try {
		localStorage.setItem(
			SNOOZE_PREFIX + userId,
			JSON.stringify({
				count,
				until: Date.now() + SNOOZE_DAYS * 24 * 60 * 60 * 1000,
			}),
		);
	} catch {
		// Storage unavailable: it will ask again next time.
	}
}

export function dismissPasskeyOffer(userId: number): void {
	try {
		localStorage.setItem(DISMISSED_PREFIX + userId, "1");
	} catch {
		// Storage unavailable: it will ask again next time.
	}
}

/**
 * `localStorage.clear()` for sign-out: "don't ask again" and the snooze are
 * per-device choices and must outlive the session.
 */
export function clearLocalStorageKeepingOffer(): void {
	try {
		const kept: [string, string][] = [];
		for (let i = 0; i < localStorage.length; i++) {
			const key = localStorage.key(i);
			const value =
				key?.startsWith(DISMISSED_PREFIX) || key?.startsWith(SNOOZE_PREFIX)
					? localStorage.getItem(key)
					: null;
			if (key && value !== null) kept.push([key, value]);
		}
		localStorage.clear();
		for (const [key, value] of kept) localStorage.setItem(key, value);
	} catch {
		// Storage unavailable: nothing to clear.
	}
}
