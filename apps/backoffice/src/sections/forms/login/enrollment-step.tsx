import { useText } from "@pengi/shared";
import { Button, Spinner } from "@pengi/ui";
import { browserSupportsWebAuthn } from "@simplewebauthn/browser";
import { ArrowLeft, Fingerprint, Smartphone } from "lucide-react";
import React from "react";
import {
	beginPasskeyEnrollment,
	beginTotpEnrollment,
	finishPasskeyEnrollment,
	finishTotpEnrollment,
	INVALID_MFA_TOKEN,
	type TotpSetup,
} from "@/api/auth-service";
import { RecoveryCodesPanel } from "@/components/features/security/recovery-codes-panel";
import { TotpScan } from "@/components/features/security/totp-scan";
import { usePasskeyCeremony } from "@/hooks/use-passkey-ceremony";
import { deviceLabel } from "@/lib/device-label";

/**
 * Forced enrollment for an admin with no second factor: choose an
 * authenticator app (QR, first code, recovery codes) or, when the API has
 * WebAuthn configured, a passkey. Finishing signs in; a TOTP enrollment waits
 * for the recovery codes to be acknowledged first (`onTotpEnrolled`). An
 * expired enrollment token goes back to the password step.
 */
export function EnrollmentStep({
	enrollmentToken,
	passkeysAvailable,
	onSignedIn,
	onTotpEnrolled,
	onBack,
	onExpired,
}: {
	enrollmentToken: string;
	passkeysAvailable: boolean;
	onSignedIn: (token: string) => void;
	onTotpEnrolled: (token: string, codes: string[]) => void;
	onBack: () => void;
	onExpired: () => void;
}) {
	const { textGet } = useText();
	const ceremony = usePasskeyCeremony();
	const [choice, setChoice] = React.useState<"choose" | "totp">("choose");
	const [setup, setSetup] = React.useState<TotpSetup>();
	const [loading, setLoading] = React.useState(false);
	const canPasskey = passkeysAvailable && browserSupportsWebAuthn();

	async function startTotp() {
		setLoading(true);
		const res = await beginTotpEnrollment(enrollmentToken);
		setLoading(false);
		if (res.success) {
			setSetup(res.data);
			setChoice("totp");
		} else if (res.data?.error_code === INVALID_MFA_TOKEN) onExpired();
	}

	async function verifyTotp(code: string) {
		setLoading(true);
		const res = await finishTotpEnrollment(enrollmentToken, code);
		setLoading(false);
		if (res.success) onTotpEnrolled(res.data.token, res.data.recovery_codes);
		else if (res.data?.error_code === INVALID_MFA_TOKEN) onExpired();
	}

	async function startPasskey() {
		setLoading(true);
		try {
			const done = await ceremony(
				() => beginPasskeyEnrollment(enrollmentToken),
				(challengeId, credential) =>
					finishPasskeyEnrollment(
						enrollmentToken,
						challengeId,
						deviceLabel() ?? textGet("auth.passkey_offer.default_name"),
						credential,
					),
			);
			if (done) onSignedIn(done.token);
		} finally {
			setLoading(false);
		}
	}

	const back = (
		<Button type="button" variant="ghost" onClick={onBack}>
			<ArrowLeft className="h-4 w-4" />
			{textGet("login.two_factor.back")}
		</Button>
	);

	return (
		<div className="w-full max-w-md min-w-xs space-y-6">
			<div className="space-y-2">
				<h1 className="text-2xl font-semibold tracking-tight">
					{textGet("backoffice.enroll.title")}
				</h1>
				<p className="text-muted-foreground">
					{textGet(
						choice === "totp"
							? "profile.two_factor.enable.scan_description"
							: "backoffice.enroll.description",
					)}
				</p>
			</div>

			{choice === "choose" && (
				<div className="space-y-3">
					<Button
						type="button"
						variant="outline"
						className="h-auto w-full justify-start gap-3 whitespace-normal p-4 text-left"
						disabled={loading}
						onClick={startTotp}
					>
						{loading ? (
							<Spinner />
						) : (
							<Smartphone className="h-5 w-5 shrink-0" />
						)}
						<span className="space-y-0.5">
							<span className="block font-medium">
								{textGet("backoffice.enroll.totp.title")}
							</span>
							<span className="block text-sm font-normal text-muted-foreground">
								{textGet("backoffice.enroll.totp.description")}
							</span>
						</span>
					</Button>
					{canPasskey && (
						<Button
							type="button"
							variant="outline"
							className="h-auto w-full justify-start gap-3 whitespace-normal p-4 text-left"
							disabled={loading}
							onClick={startPasskey}
						>
							<Fingerprint className="h-5 w-5 shrink-0" />
							<span className="space-y-0.5">
								<span className="block font-medium">
									{textGet("backoffice.enroll.passkey.title")}
								</span>
								<span className="block text-sm font-normal text-muted-foreground">
									{textGet("backoffice.enroll.passkey.description")}
								</span>
							</span>
						</Button>
					)}
					<div className="flex justify-center">{back}</div>
				</div>
			)}

			{choice === "totp" && setup && (
				<TotpScan
					setup={setup}
					loading={loading}
					onVerify={verifyTotp}
					actions={back}
				/>
			)}
		</div>
	);
}

/** The recovery codes of a first enrollment, shown once before entering. */
export function EnrollmentCodesStep({
	codes,
	onDone,
}: {
	codes: string[];
	onDone: () => void;
}) {
	const { textGet } = useText();
	return (
		<div className="w-full max-w-md min-w-xs space-y-4">
			<h1 className="text-2xl font-semibold tracking-tight">
				{textGet("profile.two_factor.codes.title")}
			</h1>
			<RecoveryCodesPanel codes={codes} onDone={onDone} />
		</div>
	);
}
