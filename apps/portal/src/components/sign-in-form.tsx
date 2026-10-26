import { useMessageStore, useText } from "@pengi/shared";
import { Button, Input, Label } from "@pengi/ui";
import React from "react";
import { requestCode, verifyCode } from "@/api/portal-service";
import { Turnstile, type TurnstileHandle } from "@/components/turnstile";
import { env } from "@/lib/env";

/**
 * Two steps: the cédula, then the 6-digit code. The copy never says whether
 * the cédula exists. `website` is a honeypot: people never see or fill it.
 * With VITE_TURNSTILE_SITE_KEY set, the code is requested with a Turnstile
 * token (one per request: the widget is reset after each one).
 */
export function SignInForm({
	slug,
	onSignedIn,
}: {
	slug: string;
	onSignedIn: (accessToken: string) => void;
}) {
	const { textGet } = useText();
	const [step, setStep] = React.useState<"document" | "code">("document");
	const [document, setDocument] = React.useState("");
	const [website, setWebsite] = React.useState("");
	const [code, setCode] = React.useState("");
	const [busy, setBusy] = React.useState(false);
	const [error, setError] = React.useState<string>();
	const siteKey = env.VITE_TURNSTILE_SITE_KEY;
	const language = useMessageStore((state) => state.lang) ?? "es";
	const captcha = React.useRef<TurnstileHandle>(null);
	const [captchaToken, setCaptchaToken] = React.useState<string>();
	const [captchaFailed, setCaptchaFailed] = React.useState(false);
	const onCaptchaToken = React.useCallback((token: string | undefined) => {
		setCaptchaToken(token);
		if (token) setCaptchaFailed(false);
	}, []);
	const onCaptchaError = React.useCallback(() => setCaptchaFailed(true), []);

	const submitDocument = async (event: React.FormEvent) => {
		event.preventDefault();
		if (siteKey && !captchaToken) return;
		setBusy(true);
		const res = await requestCode(slug, {
			document: document.trim(),
			website,
			...(siteKey ? { captcha_token: captchaToken } : {}),
		});
		// A token works once: get a fresh one for the next request.
		captcha.current?.reset();
		setBusy(false);
		if (res.success) {
			setError(undefined);
			setStep("code");
		}
	};

	const submitCode = async (event: React.FormEvent) => {
		event.preventDefault();
		setBusy(true);
		const res = await verifyCode(slug, document.trim(), code);
		setBusy(false);
		if (res.success) {
			onSignedIn(res.data.access_token);
			return;
		}
		setError(
			res.code === 401 || res.code === 400
				? textGet("portal.signin.code_invalid")
				: res.message,
		);
	};

	if (step === "code") {
		return (
			<form onSubmit={submitCode} className="space-y-4">
				<p className="text-muted-foreground text-sm">
					{textGet("portal.signin.code_sent")}
				</p>
				<div className="space-y-2">
					<Label htmlFor="portal-code">{textGet("portal.signin.code")}</Label>
					<Input
						id="portal-code"
						name="code"
						inputMode="numeric"
						autoComplete="one-time-code"
						maxLength={6}
						value={code}
						onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
						aria-invalid={error ? true : undefined}
						className="h-11 text-center text-xl tracking-widest"
					/>
					{error && (
						<p role="alert" className="text-destructive text-sm">
							{error}
						</p>
					)}
				</div>
				<Button
					type="submit"
					className="h-11 w-full"
					disabled={busy || code.length !== 6}
				>
					{textGet("portal.signin.verify")}
				</Button>
				<Button
					type="button"
					variant="ghost"
					className="w-full"
					onClick={() => {
						setCode("");
						setError(undefined);
						setStep("document");
					}}
				>
					{textGet("portal.signin.back")}
				</Button>
			</form>
		);
	}

	return (
		<form onSubmit={submitDocument} className="space-y-4">
			<div className="space-y-2">
				<Label htmlFor="portal-document">
					{textGet("portal.signin.document")}
				</Label>
				<Input
					id="portal-document"
					name="document"
					inputMode="numeric"
					autoComplete="off"
					value={document}
					onChange={(e) => setDocument(e.target.value)}
					className="h-11"
				/>
			</div>
			{/* Honeypot: off screen, skipped by keyboard and screen readers. */}
			<div
				aria-hidden="true"
				className="absolute -left-[9999px] h-0 w-0 overflow-hidden opacity-0"
			>
				<input
					type="text"
					name="website"
					tabIndex={-1}
					autoComplete="off"
					value={website}
					onChange={(e) => setWebsite(e.target.value)}
				/>
			</div>
			{siteKey && (
				<div className="space-y-2">
					<Turnstile
						ref={captcha}
						siteKey={siteKey}
						language={language}
						onToken={onCaptchaToken}
						onError={onCaptchaError}
					/>
					{captchaFailed && (
						<p role="alert" className="text-destructive text-sm">
							{textGet("portal.signin.captcha_error")}
						</p>
					)}
				</div>
			)}
			<Button
				type="submit"
				className="h-11 w-full"
				disabled={
					busy || document.trim() === "" || (!!siteKey && !captchaToken)
				}
			>
				{textGet("portal.signin.send_code")}
			</Button>
			<p className="text-muted-foreground text-xs">
				{textGet("portal.signin.hint")}
			</p>
		</form>
	);
}
