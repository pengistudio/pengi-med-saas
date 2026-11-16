import { useText } from "@pengi/shared";
import { Button, Spinner, useToast } from "@pengi/ui";
import {
	browserSupportsWebAuthn,
	startAuthentication,
} from "@simplewebauthn/browser";
import { KeyRound } from "lucide-react";
import React from "react";
import { beginPasskeyLogin, loginWithPasskey } from "@/api/auth-service";

/**
 * "Sign in with a passkey": begin -> browser ceremony -> login. A passkey is
 * a complete sign-in (no TOTP). Hands over the session token. Hidden where
 * WebAuthn is missing; a cancelled prompt (NotAllowedError) is silent.
 */
export function PasskeyLoginButton({
	onSuccess,
	disabled,
}: {
	onSuccess: (token: string) => void;
	disabled?: boolean;
}) {
	const { textGet } = useText();
	const { errorToast } = useToast();
	const [load, setLoad] = React.useState(false);

	if (!browserSupportsWebAuthn()) return null;

	async function onClick() {
		setLoad(true);
		try {
			const begin = await beginPasskeyLogin();
			if (!begin.success) return;
			let credential: Awaited<ReturnType<typeof startAuthentication>>;
			try {
				credential = await startAuthentication({
					optionsJSON: begin.data.options,
				});
			} catch (err) {
				if (!(err instanceof Error && err.name === "NotAllowedError")) {
					errorToast(null, textGet("auth.passkey.browser_error"));
				}
				return;
			}
			const res = await loginWithPasskey(begin.data.challenge_id, credential);
			if (res.success) onSuccess(res.data.token);
		} finally {
			setLoad(false);
		}
	}

	return (
		<Button
			type="button"
			variant="outline"
			className="w-full"
			disabled={load || disabled}
			onClick={onClick}
		>
			{load ? <Spinner /> : <KeyRound className="h-4 w-4" />}
			{textGet("auth.passkey.login_button")}
		</Button>
	);
}
