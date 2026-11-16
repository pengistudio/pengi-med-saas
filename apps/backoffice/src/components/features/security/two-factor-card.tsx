import { useText } from "@pengi/shared";
import {
	Badge,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Form,
	FormInput,
	FormPasswordInput,
	Spinner,
} from "@pengi/ui";
import { KeyRound, ShieldCheck, ShieldOff } from "lucide-react";
import React from "react";
import z from "zod";
import type { TotpSetup } from "@/api/auth-service";
import {
	disableTwoFactor,
	enableTwoFactor,
	getTwoFactorStatus,
	regenerateRecoveryCodes,
	setupTwoFactor,
	type TwoFactorStatus,
} from "@/api/two-factor-service";
import { RecoveryCodesPanel } from "./recovery-codes-panel";
import { TotpScan } from "./totp-scan";

/** At or below this many unused recovery codes, suggest a new set. */
const LOW_RECOVERY_CODES = 3;

// Field names (= input ids) differ from the other dialogs on the page: the
// labels must point at their inputs.
const passwordSchema = z.object({
	two_factor_password: z.string().min(1),
});

const proofSchema = z.object({
	two_factor_password: z.string().min(1),
	two_factor_code: z.string().trim().min(6).max(64),
});

type DialogKind = "enable" | "disable" | "regenerate";

const PROOF_COPY = {
	disable: {
		title: "profile.two_factor.disable.title",
		description: "profile.two_factor.disable.description",
		submit: "profile.two_factor.disable.submit",
	},
	regenerate: {
		title: "profile.two_factor.regenerate.title",
		description: "profile.two_factor.regenerate.description",
		submit: "profile.two_factor.regenerate.submit",
	},
} as const;

/**
 * Authenticator-app 2FA of the signed-in admin: status, enrollment and, once
 * on, new recovery codes and disable. The last second factor can't be
 * removed: without a passkey, disable is blocked.
 */
export function TwoFactorCard({
	version = 0,
}: {
	/** Bump to reload the status (the passkeys changed). */
	version?: number;
}) {
	const { textGet } = useText();
	const [status, setStatus] = React.useState<TwoFactorStatus>();
	const [dialog, setDialog] = React.useState<DialogKind | null>(null);

	const loadStatus = React.useCallback(() => {
		getTwoFactorStatus().then((res) => {
			if (res.success) setStatus(res.data);
		});
	}, []);

	React.useEffect(() => {
		loadStatus();
	}, [loadStatus, version]);

	const close = () => setDialog(null);
	const remaining = status?.recovery_codes_remaining ?? 0;
	const lastFactor = status?.enabled === true && status.passkey_count === 0;

	return (
		<Card>
			<CardHeader>
				<CardTitle className="flex flex-wrap items-center gap-2">
					{textGet("profile.two_factor.title")}
					{status && (
						<Badge variant={status.enabled ? "default" : "outline"}>
							{textGet(
								status.enabled
									? "profile.two_factor.status.enabled"
									: "profile.two_factor.status.disabled",
							)}
						</Badge>
					)}
				</CardTitle>
				<CardDescription>
					{textGet("backoffice.security.two_factor.description")}
				</CardDescription>
			</CardHeader>
			<CardContent className="space-y-4">
				{!status ? (
					<Spinner />
				) : status.enabled ? (
					<>
						<p className="text-sm text-muted-foreground">
							{textGet("profile.two_factor.recovery_remaining", {
								count: remaining,
							})}
						</p>
						{remaining <= LOW_RECOVERY_CODES && (
							<p role="alert" className="text-sm text-destructive">
								{textGet("profile.two_factor.recovery_low")}
							</p>
						)}
						{lastFactor && (
							<p className="text-sm text-muted-foreground">
								{textGet("backoffice.security.disable_blocked")}
							</p>
						)}
						<div className="flex flex-wrap justify-end gap-2">
							<Button variant="outline" onClick={() => setDialog("regenerate")}>
								<KeyRound className="h-4 w-4" />
								{textGet("profile.two_factor.regenerate")}
							</Button>
							<Button
								variant="destructive"
								disabled={lastFactor}
								onClick={() => setDialog("disable")}
							>
								<ShieldOff className="h-4 w-4" />
								{textGet("profile.two_factor.disable")}
							</Button>
						</div>
					</>
				) : (
					<div className="flex justify-end">
						<Button onClick={() => setDialog("enable")}>
							<ShieldCheck className="h-4 w-4" />
							{textGet("profile.two_factor.enable")}
						</Button>
					</div>
				)}
			</CardContent>

			{dialog === "enable" && (
				<EnableDialog onClose={close} onChanged={loadStatus} />
			)}
			{(dialog === "disable" || dialog === "regenerate") && (
				<ProofDialog kind={dialog} onClose={close} onChanged={loadStatus} />
			)}
		</Card>
	);
}

/** Enrollment: current password -> QR + secret -> first code -> recovery codes. */
function EnableDialog({
	onClose,
	onChanged,
}: {
	onClose: () => void;
	onChanged: () => void;
}) {
	const { textGet } = useText();
	const [step, setStep] = React.useState<"password" | "scan" | "codes">(
		"password",
	);
	const [setup, setSetup] = React.useState<TotpSetup>();
	const [codes, setCodes] = React.useState<string[]>([]);
	const [loading, setLoading] = React.useState(false);

	async function onPassword(values: z.infer<typeof passwordSchema>) {
		setLoading(true);
		const res = await setupTwoFactor(values.two_factor_password);
		setLoading(false);
		if (res.success) {
			setSetup(res.data);
			setStep("scan");
		}
	}

	async function onVerify(code: string) {
		setLoading(true);
		const res = await enableTwoFactor(code);
		setLoading(false);
		if (!res.success) return;
		setCodes(res.data.recovery_codes);
		setStep("codes");
		onChanged();
	}

	return (
		<Dialog
			open
			onOpenChange={(open) => {
				// The codes are shown once: leave only through "I saved them".
				if (!open && step !== "codes") onClose();
			}}
		>
			<DialogContent className="sm:max-w-md" showCloseButton={step !== "codes"}>
				<DialogHeader>
					<DialogTitle>
						{textGet(
							step === "codes"
								? "profile.two_factor.codes.title"
								: "profile.two_factor.enable.title",
						)}
					</DialogTitle>
					{step !== "codes" && (
						<DialogDescription>
							{textGet(
								step === "password"
									? "profile.two_factor.enable.password_description"
									: "profile.two_factor.enable.scan_description",
							)}
						</DialogDescription>
					)}
				</DialogHeader>

				{step === "password" && (
					<Form<typeof passwordSchema>
						schema={passwordSchema}
						onSubmit={onPassword}
						defaultValues={{ two_factor_password: "" }}
					>
						{(field) => (
							<div className="space-y-4">
								<FormPasswordInput
									field={field}
									name="two_factor_password"
									label={textGet("profile.security.current_password")}
									autoFocus
								/>
								<DialogFooter>
									<Button type="button" variant="outline" onClick={onClose}>
										{textGet("profile.two_factor.cancel")}
									</Button>
									<Button type="submit" disabled={loading}>
										{loading && <Spinner />}
										{textGet("profile.two_factor.continue")}
									</Button>
								</DialogFooter>
							</div>
						)}
					</Form>
				)}

				{step === "scan" && setup && (
					<TotpScan
						setup={setup}
						loading={loading}
						onVerify={onVerify}
						actions={
							<Button type="button" variant="outline" onClick={onClose}>
								{textGet("profile.two_factor.cancel")}
							</Button>
						}
					/>
				)}

				{step === "codes" && (
					<RecoveryCodesPanel
						codes={codes}
						onDone={() => {
							setCodes([]);
							onClose();
						}}
					/>
				)}
			</DialogContent>
		</Dialog>
	);
}

/**
 * Disable, or a new set of recovery codes: both ask for the current password
 * and a code (authenticator or recovery). A wrong one answers 422 and the
 * service shows why; the session is untouched.
 */
function ProofDialog({
	kind,
	onClose,
	onChanged,
}: {
	kind: "disable" | "regenerate";
	onClose: () => void;
	onChanged: () => void;
}) {
	const { textGet } = useText();
	const [loading, setLoading] = React.useState(false);
	const [codes, setCodes] = React.useState<string[]>([]);

	async function onSubmit(values: z.infer<typeof proofSchema>) {
		setLoading(true);
		if (kind === "disable") {
			const res = await disableTwoFactor(
				values.two_factor_password,
				values.two_factor_code,
			);
			setLoading(false);
			if (res.success) {
				onChanged();
				onClose();
			}
			return;
		}
		const res = await regenerateRecoveryCodes(
			values.two_factor_password,
			values.two_factor_code,
		);
		setLoading(false);
		if (res.success) {
			setCodes(res.data.recovery_codes);
			onChanged();
		}
	}

	const showingCodes = codes.length > 0;
	const copy = PROOF_COPY[kind];

	return (
		<Dialog
			open
			onOpenChange={(open) => {
				if (!open && !showingCodes) onClose();
			}}
		>
			<DialogContent className="sm:max-w-md" showCloseButton={!showingCodes}>
				<DialogHeader>
					<DialogTitle>
						{textGet(
							showingCodes ? "profile.two_factor.codes.title" : copy.title,
						)}
					</DialogTitle>
					{!showingCodes && (
						<DialogDescription>{textGet(copy.description)}</DialogDescription>
					)}
				</DialogHeader>
				{showingCodes ? (
					<RecoveryCodesPanel
						codes={codes}
						onDone={() => {
							setCodes([]);
							onClose();
						}}
					/>
				) : (
					<Form<typeof proofSchema>
						schema={proofSchema}
						onSubmit={onSubmit}
						defaultValues={{ two_factor_password: "", two_factor_code: "" }}
					>
						{(field) => (
							<div className="space-y-4">
								<FormPasswordInput
									field={field}
									name="two_factor_password"
									label={textGet("profile.security.current_password")}
									autoFocus
								/>
								<FormInput
									field={field}
									name="two_factor_code"
									type="text"
									label={textGet("profile.two_factor.proof.code_label")}
									description={textGet(
										"profile.two_factor.proof.code_description",
									)}
									autoComplete="one-time-code"
									spellCheck={false}
								/>
								<DialogFooter>
									<Button type="button" variant="outline" onClick={onClose}>
										{textGet("profile.two_factor.cancel")}
									</Button>
									<Button
										type="submit"
										variant={kind === "disable" ? "destructive" : "default"}
										disabled={loading}
									>
										{loading && <Spinner />}
										{textGet(copy.submit)}
									</Button>
								</DialogFooter>
							</div>
						)}
					</Form>
				)}
			</DialogContent>
		</Dialog>
	);
}
