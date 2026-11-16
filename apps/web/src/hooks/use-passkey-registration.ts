import { useText } from "@pengi/shared";
import { useToast } from "@pengi/ui";
import { startRegistration } from "@simplewebauthn/browser";
import {
	beginPasskeyRegistration,
	finishPasskeyRegistration,
	PASSKEY_REAUTH_REQUIRED,
} from "@/api/passkey-service";

/**
 * done: registered. reauth: the session is not fresh, ask for the password
 * and retry. stopped: cancelled, failed or refused (already reported).
 */
export type PasskeyRegistrationResult = "done" | "reauth" | "stopped";

export interface PasskeyRegistrationInput {
	name: string;
	/** Omitted while the session is fresh; `code` goes with it when 2FA is on. */
	password?: string;
	code?: string;
}

/**
 * Registration ceremony shared by the profile card and the post-login offer:
 * begin -> browser prompt -> finish. A cancelled prompt (NotAllowedError) is
 * silent; other browser errors toast auth.passkey.browser_error. The services
 * own every other toast.
 */
export function usePasskeyRegistration() {
	const { textGet } = useText();
	const { errorToast } = useToast();

	return async function register({
		name,
		password,
		code,
	}: PasskeyRegistrationInput): Promise<PasskeyRegistrationResult> {
		const begin = await beginPasskeyRegistration(password, code);
		if (!begin.success) {
			return !password && begin.data?.error_code === PASSKEY_REAUTH_REQUIRED
				? "reauth"
				: "stopped";
		}
		let credential: Awaited<ReturnType<typeof startRegistration>>;
		try {
			credential = await startRegistration({
				optionsJSON: begin.data.options,
			});
		} catch (err) {
			if (!(err instanceof Error && err.name === "NotAllowedError")) {
				errorToast(null, textGet("auth.passkey.browser_error"));
			}
			return "stopped";
		}
		const res = await finishPasskeyRegistration(
			begin.data.challenge_id,
			name,
			credential,
		);
		return res.success ? "done" : "stopped";
	};
}
