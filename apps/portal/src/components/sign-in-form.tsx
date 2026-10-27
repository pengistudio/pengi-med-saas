import { useText } from "@pengi/shared";
import { Button, Input, Label } from "@pengi/ui";
import React from "react";
import { requestCode, verifyCode } from "@/api/portal-service";
import {
	CaptchaField,
	CodeInput,
	Honeypot,
	useCaptcha,
} from "@/components/verification";

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
	const captcha = useCaptcha();

	const submitDocument = async (event: React.FormEvent) => {
		event.preventDefault();
		if (captcha.pending) return;
		setBusy(true);
		const res = await requestCode(slug, {
			document: document.trim(),
			website,
			...captcha.payload(),
		});
		// A token works once: get a fresh one for the next request.
		captcha.reset();
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
				<CodeInput
					id="portal-code"
					value={code}
					onChange={setCode}
					error={error}
				/>
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
			<Honeypot value={website} onChange={setWebsite} />
			<CaptchaField captcha={captcha} />
			<Button
				type="submit"
				className="h-11 w-full"
				disabled={busy || document.trim() === "" || captcha.pending}
			>
				{textGet("portal.signin.send_code")}
			</Button>
			<p className="text-muted-foreground text-xs">
				{textGet("portal.signin.hint")}
			</p>
		</form>
	);
}
