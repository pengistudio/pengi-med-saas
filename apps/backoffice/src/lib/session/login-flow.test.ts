import { describe, expect, it } from "vitest";
import { initialStep, nextStep } from "./login-flow";

describe("login flow", () => {
	it("goes to the second factor with the mfa token and what the admin has", () => {
		const state = nextStep(initialStep, {
			type: "password",
			outcome: {
				kind: "two_factor",
				mfaToken: "m",
				totp: false,
				passkey: true,
			},
		});
		expect(state).toEqual({
			step: "second_factor",
			mfaToken: "m",
			totp: false,
			passkey: true,
		});
	});

	it("goes to enrollment when the admin has no second factor", () => {
		const state = nextStep(initialStep, {
			type: "password",
			outcome: {
				kind: "enrollment",
				enrollmentToken: "e",
				passkeysAvailable: false,
			},
		});
		expect(state).toEqual({
			step: "enroll",
			enrollmentToken: "e",
			passkeysAvailable: false,
		});
	});

	it("stays on the password step when the login failed", () => {
		expect(
			nextStep(initialStep, { type: "password", outcome: { kind: "failed" } }),
		).toBe(initialStep);
	});

	it("returns to the password step when the token expired or on back", () => {
		const second = nextStep(initialStep, {
			type: "password",
			outcome: {
				kind: "two_factor",
				mfaToken: "m",
				totp: true,
				passkey: false,
			},
		});
		expect(nextStep(second, { type: "expired" })).toEqual(initialStep);
		expect(nextStep(second, { type: "back" })).toEqual(initialStep);
	});

	it("holds the session token until the recovery codes are acknowledged", () => {
		const enroll = nextStep(initialStep, {
			type: "password",
			outcome: {
				kind: "enrollment",
				enrollmentToken: "e",
				passkeysAvailable: true,
			},
		});
		expect(
			nextStep(enroll, {
				type: "totp_enrolled",
				token: "t",
				codes: ["a", "b"],
			}),
		).toEqual({ step: "recovery_codes", token: "t", codes: ["a", "b"] });
	});

	it("ignores an enrollment result outside the enrollment step", () => {
		expect(
			nextStep(initialStep, { type: "totp_enrolled", token: "t", codes: [] }),
		).toBe(initialStep);
	});
});
