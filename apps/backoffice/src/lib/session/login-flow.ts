import type { LoginOutcome } from "./session";

/**
 * The screens of a backoffice sign-in. Tokens live only in this state (never
 * persisted): the mfa and enrollment tokens are short-lived, and the session
 * token of a first-time enrollment waits here until the admin confirms they
 * saved the recovery codes.
 */
export type LoginStep =
	| { step: "password" }
	| { step: "second_factor"; mfaToken: string; totp: boolean; passkey: boolean }
	| { step: "enroll"; enrollmentToken: string; passkeysAvailable: boolean }
	| { step: "recovery_codes"; token: string; codes: string[] };

export type LoginEvent =
	/** The password step answered. */
	| { type: "password"; outcome: LoginOutcome }
	/** The mfa/enrollment token expired or no longer matches: start over. */
	| { type: "expired" }
	| { type: "back" }
	/** TOTP enrollment finished: the session is held until the codes are saved. */
	| { type: "totp_enrolled"; token: string; codes: string[] };

export const initialStep: LoginStep = { step: "password" };

export function nextStep(state: LoginStep, event: LoginEvent): LoginStep {
	switch (event.type) {
		case "password": {
			const { outcome } = event;
			if (outcome.kind === "two_factor") {
				return {
					step: "second_factor",
					mfaToken: outcome.mfaToken,
					totp: outcome.totp,
					passkey: outcome.passkey,
				};
			}
			if (outcome.kind === "enrollment") {
				return {
					step: "enroll",
					enrollmentToken: outcome.enrollmentToken,
					passkeysAvailable: outcome.passkeysAvailable,
				};
			}
			return state;
		}
		case "expired":
		case "back":
			return initialStep;
		case "totp_enrolled":
			return state.step === "enroll"
				? { step: "recovery_codes", token: event.token, codes: event.codes }
				: state;
	}
}
