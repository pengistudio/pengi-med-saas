import { useText } from "@pengi/shared";
import {
	Button,
	Form,
	FormInput,
	Label,
	OtpInput,
	Spinner,
	Text,
} from "@pengi/ui";
import { ArrowLeft } from "lucide-react";
import React from "react";
import z from "zod";
import { INVALID_MFA_TOKEN, loginTwoFactor } from "@/api/auth-service";
import { PasskeyLoginButton } from "./passkey-login-button";

const TOTP_LENGTH = 6;

const recoverySchema = z.object({
	code: z.string().trim().min(8).max(64),
});

/**
 * Second step of a sign-in: a 6-digit code from the authenticator app or a
 * recovery code, or a passkey when the admin has one. An admin with only
 * passkeys sees just the passkey button. The mfa token lives only in the
 * caller's state; an expired one sends the admin back to the password step.
 */
export function TwoFactorStep({
	mfaToken,
	totp,
	passkey,
	onSuccess,
	onBack,
	onExpired,
}: {
	mfaToken: string;
	totp: boolean;
	passkey: boolean;
	onSuccess: (token: string) => void;
	onBack: () => void;
	onExpired: () => void;
}) {
	const { textGet } = useText();
	const [mode, setMode] = React.useState<"app" | "recovery">("app");
	const [code, setCode] = React.useState("");
	const [load, setLoad] = React.useState(false);

	async function submit(value: string) {
		setLoad(true);
		const res = await loginTwoFactor(mfaToken, value);
		setLoad(false);
		if (res.success) {
			onSuccess(res.data.token);
			return;
		}
		// Wrong code: the service shows why; let the admin try again.
		setCode("");
		if (res.data?.error_code === INVALID_MFA_TOKEN) onExpired();
	}

	function onAppSubmit(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		if (code.length === TOTP_LENGTH && !load) submit(code);
	}

	return (
		<div className="w-full max-w-md min-w-xs space-y-6">
			<div className="space-y-2">
				<h1 className="text-2xl font-semibold tracking-tight">
					<Text uuid="login.two_factor.title" />
				</h1>
				{totp && (
					<p className="text-muted-foreground">
						<Text
							uuid={
								mode === "app"
									? "login.two_factor.description"
									: "login.two_factor.recovery_description"
							}
						/>
					</p>
				)}
			</div>

			{totp &&
				(mode === "app" ? (
					<form onSubmit={onAppSubmit} className="space-y-6">
						<div className="space-y-2">
							<Label htmlFor="two-factor-code">
								{textGet("login.two_factor.code_label")}
							</Label>
							<OtpInput
								id="two-factor-code"
								name="code"
								length={TOTP_LENGTH}
								value={code}
								onValueChange={setCode}
								autoFocus
								className="[&>input]:flex-1"
							/>
						</div>
						<Button
							type="submit"
							className="w-full"
							disabled={load || code.length !== TOTP_LENGTH}
						>
							{load && <Spinner />}
							<Text uuid="login.two_factor.submit" />
						</Button>
					</form>
				) : (
					<Form<typeof recoverySchema>
						schema={recoverySchema}
						onSubmit={(values) => submit(values.code)}
						defaultValues={{ code: "" }}
					>
						{(field) => (
							<div className="space-y-6">
								<FormInput
									field={field}
									name="code"
									type="text"
									label={textGet("login.two_factor.recovery_label")}
									placeholder={textGet("login.two_factor.recovery_placeholder")}
									autoComplete="off"
									autoCapitalize="characters"
									spellCheck={false}
									autoFocus
								/>
								<Button type="submit" className="w-full" disabled={load}>
									{load && <Spinner />}
									<Text uuid="login.two_factor.submit" />
								</Button>
							</div>
						)}
					</Form>
				))}

			{passkey && <PasskeyLoginButton onSuccess={onSuccess} disabled={load} />}

			<div className="flex flex-col items-center gap-2 text-sm">
				{totp && (
					<Button
						type="button"
						variant="link"
						onClick={() => {
							setCode("");
							setMode(mode === "app" ? "recovery" : "app");
						}}
					>
						<Text
							uuid={
								mode === "app"
									? "login.two_factor.use_recovery"
									: "login.two_factor.use_app"
							}
						/>
					</Button>
				)}
				<Button type="button" variant="ghost" onClick={onBack}>
					<ArrowLeft className="h-4 w-4" />
					<Text uuid="login.two_factor.back" />
				</Button>
			</div>
		</div>
	);
}
