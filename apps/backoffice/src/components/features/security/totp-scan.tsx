import { useText } from "@pengi/shared";
import { Button, Label, OtpInput, Spinner } from "@pengi/ui";
import { Check, Copy } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";
import React from "react";
import type { TotpSetup } from "@/api/auth-service";

export const TOTP_LENGTH = 6;

/**
 * Authenticator enrollment: the QR (and the secret for manual entry) plus the
 * first code. `actions` sits next to the verify button (cancel / back).
 */
export function TotpScan({
	setup,
	loading,
	onVerify,
	actions,
}: {
	setup: TotpSetup;
	loading: boolean;
	onVerify: (code: string) => void | Promise<void>;
	actions?: React.ReactNode;
}) {
	const { textGet } = useText();
	const [code, setCode] = React.useState("");

	async function submit(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		if (code.length !== TOTP_LENGTH || loading) return;
		await onVerify(code);
		setCode("");
	}

	return (
		<form onSubmit={submit} className="space-y-4">
			<div className="flex justify-center">
				<div className="rounded-lg bg-white p-3">
					<QRCodeSVG
						value={setup.otpauth_uri}
						size={176}
						title={textGet("profile.two_factor.enable.qr_title")}
					/>
				</div>
			</div>
			<SecretText secret={setup.secret} />
			<div className="space-y-2">
				<Label htmlFor="totp-scan-code">
					{textGet("profile.two_factor.code_label")}
				</Label>
				<OtpInput
					id="totp-scan-code"
					name="code"
					length={TOTP_LENGTH}
					value={code}
					onValueChange={setCode}
					className="[&>input]:flex-1"
				/>
			</div>
			<div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
				{actions}
				<Button type="submit" disabled={loading || code.length !== TOTP_LENGTH}>
					{loading && <Spinner />}
					{textGet("profile.two_factor.verify")}
				</Button>
			</div>
		</form>
	);
}

/** The secret for manual entry, in groups of four, with a copy button. */
function SecretText({ secret }: { secret: string }) {
	const { textGet } = useText();
	const [copied, setCopied] = React.useState(false);

	async function copy() {
		await navigator.clipboard.writeText(secret);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	}

	return (
		<div className="space-y-1.5">
			<p className="text-sm font-medium">
				{textGet("profile.two_factor.enable.secret_label")}
			</p>
			<div className="flex items-center gap-2">
				<code
					data-testid="two-factor-secret"
					data-secret={secret}
					className="min-w-0 flex-1 break-all rounded-md border bg-muted px-3 py-2 font-mono text-sm"
				>
					{secret.match(/.{1,4}/g)?.join(" ")}
				</code>
				<Button
					type="button"
					variant="outline"
					size="icon"
					onClick={copy}
					aria-label={textGet("profile.two_factor.enable.copy_secret")}
				>
					{copied ? (
						<Check className="h-4 w-4" />
					) : (
						<Copy className="h-4 w-4" />
					)}
				</Button>
			</div>
		</div>
	);
}
