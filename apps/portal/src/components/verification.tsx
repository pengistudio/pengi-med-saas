import { useMessageStore, useText } from "@pengi/shared";
import { Label, OtpInput } from "@pengi/ui";
import React from "react";
import { Turnstile, type TurnstileHandle } from "@/components/turnstile";
import { env } from "@/lib/env";

/**
 * The pieces of asking for a code (sign-in and new patients): the Turnstile
 * captcha, the honeypot and the 6-digit code field.
 */

/**
 * Captcha state. With VITE_TURNSTILE_SITE_KEY set, a code is requested with a
 * Turnstile token; tokens work once, so `reset()` after each request.
 */
export function useCaptcha() {
	const siteKey = env.VITE_TURNSTILE_SITE_KEY;
	const ref = React.useRef<TurnstileHandle>(null);
	const [token, setToken] = React.useState<string>();
	const [failed, setFailed] = React.useState(false);
	const onToken = React.useCallback((value: string | undefined) => {
		setToken(value);
		if (value) setFailed(false);
	}, []);
	const onError = React.useCallback(() => setFailed(true), []);

	return {
		siteKey,
		ref,
		token,
		failed,
		onToken,
		onError,
		/** Whether the form must wait for the captcha before sending. */
		pending: !!siteKey && !token,
		/** The request's captcha field (nothing without a site key). */
		payload: () => (siteKey ? { captcha_token: token } : {}),
		reset: () => ref.current?.reset(),
	};
}

export type Captcha = ReturnType<typeof useCaptcha>;

/** The Turnstile widget and its error, when a site key is set. */
export function CaptchaField({ captcha }: { captcha: Captcha }) {
	const { textGet } = useText();
	const language = useMessageStore((state) => state.lang) ?? "es";
	if (!captcha.siteKey) return null;
	return (
		<div className="space-y-2">
			<Turnstile
				ref={captcha.ref}
				siteKey={captcha.siteKey}
				language={language}
				onToken={captcha.onToken}
				onError={captcha.onError}
			/>
			{captcha.failed && (
				<p role="alert" className="text-destructive text-sm">
					{textGet("portal.signin.captcha_error")}
				</p>
			)}
		</div>
	);
}

/** Honeypot: off screen, skipped by keyboard and screen readers. People never fill it. */
export function Honeypot({
	value,
	onChange,
}: {
	value: string;
	onChange: (value: string) => void;
}) {
	return (
		<div
			aria-hidden="true"
			className="absolute -left-[9999px] h-0 w-0 overflow-hidden opacity-0"
		>
			<input
				type="text"
				name="website"
				tabIndex={-1}
				autoComplete="off"
				value={value}
				onChange={(e) => onChange(e.target.value)}
			/>
		</div>
	);
}

/** The 6-digit code, one slot per digit, with its error under it. */
export function CodeInput({
	id,
	value,
	onChange,
	error,
}: {
	id: string;
	value: string;
	onChange: (value: string) => void;
	error?: string;
}) {
	const { textGet } = useText();
	return (
		<div className="space-y-2">
			<Label htmlFor={id}>{textGet("portal.signin.code")}</Label>
			<OtpInput
				id={id}
				name="code"
				length={6}
				value={value}
				onValueChange={onChange}
				aria-invalid={error ? true : undefined}
				className="[&>input]:flex-1"
			/>
			{error && (
				<p role="alert" className="text-destructive text-sm">
					{error}
				</p>
			)}
		</div>
	);
}
