import { type ServiceResponse, useText } from "@pengi/shared";
import { useToast } from "@pengi/ui";
import { startRegistration } from "@simplewebauthn/browser";
import type { PasskeyRegistrationBegin } from "@/api/auth-service";

/**
 * The browser side of registering a passkey: begin -> prompt -> finish. The
 * caller supplies the two API calls (signed-in or enrollment flavour); the
 * services own every toast but the browser's own errors. A cancelled prompt
 * (NotAllowedError) is silent. Resolves null when anything stopped.
 */
export function usePasskeyCeremony() {
	const { textGet } = useText();
	const { errorToast } = useToast();

	return async function register<T>(
		begin: () => Promise<ServiceResponse<PasskeyRegistrationBegin>>,
		finish: (
			challengeId: string,
			credential: Awaited<ReturnType<typeof startRegistration>>,
		) => Promise<ServiceResponse<T>>,
	): Promise<T | null> {
		const started = await begin();
		if (!started.success) return null;
		let credential: Awaited<ReturnType<typeof startRegistration>>;
		try {
			credential = await startRegistration({
				optionsJSON: started.data.options,
			});
		} catch (err) {
			if (!(err instanceof Error && err.name === "NotAllowedError")) {
				errorToast(null, textGet("auth.passkey.browser_error"));
			}
			return null;
		}
		const res = await finish(started.data.challenge_id, credential);
		return res.success ? res.data : null;
	};
}
