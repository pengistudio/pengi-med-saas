import { useText } from "@pengi/shared";
import {
	Button,
	Form,
	FormInput,
	FormPasswordInput,
	Spinner,
	Text,
} from "@pengi/ui";
import React from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import z from "zod";
import { userLogin } from "@/api/auth-service";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { LANDING_URL } from "@/lib/constants";
import { useTokenStore } from "@/store/token-store";
import { isTwoFactorChallenge, type LoginResponse } from "@/types/user-type";
import { PasskeyLoginButton } from "./passkey-login-button";
import { TwoFactorStep } from "./two-factor-step";

const formSchema = z.object({
	password: z
		.string()
		.min(6)
		.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
	user_name: z
		.string()
		.min(3)
		.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
});

const LoginForm = () => {
	const { textGet } = useText();
	const [load, setLoad] = React.useState(false);
	const [emailNotVerified, setEmailNotVerified] = React.useState(false);
	const [capsLock, setCapsLock] = React.useState(false);
	const { setToken } = useTokenStore();
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const intent = searchParams.get("intent");
	// Only in memory: a reload or "back" returns to the password step.
	const [mfaToken, setMfaToken] = React.useState<string>();

	if (mfaToken) {
		return (
			<TwoFactorStep
				mfaToken={mfaToken}
				onSuccess={finishLogin}
				onBack={() => setMfaToken(undefined)}
			/>
		);
	}

	return (
		<Form<typeof formSchema>
			schema={formSchema}
			onSubmit={onSubmit}
			className="w-full"
			defaultValues={{
				password: "",
				user_name: "",
			}}
		>
			{(field) => (
				<div className="space-y-6">
					<div className="space-y-2">
						<h1 className="text-3xl font-semibold tracking-tight">
							<Text
								uuid={
									intent === "new_company"
										? "login.new_company.title"
										: "login.title"
								}
							/>
						</h1>
						{intent === "new_company" && (
							<p className="text-muted-foreground">
								<Text uuid="login.new_company.subtitle" />
							</p>
						)}
					</div>
					{emailNotVerified && (
						<Alert variant="destructive">
							<AlertTitle>
								<Text uuid="login.email_not_verified.title" />
							</AlertTitle>
							<AlertDescription>
								<Text uuid="login.email_not_verified.description" />
							</AlertDescription>
						</Alert>
					)}
					<div className="space-y-4">
						<FormInput
							field={field}
							name="user_name"
							type="text"
							label={textGet("login.username")}
							autoComplete="username"
							autoFocus
						/>
						<div className="relative">
							<FormPasswordInput
								field={field}
								name="password"
								label={textGet("login.password")}
								description={capsLock ? textGet("login.caps_lock") : undefined}
								onKeyDown={detectCapsLock}
								onKeyUp={detectCapsLock}
							/>
							<Link
								to="/forgot-password"
								className="absolute top-0 right-0 text-sm text-primary hover:underline underline-offset-4"
							>
								<Text uuid="login.forgot_password" />
							</Link>
						</div>
					</div>
					<Button type="submit" size="lg" className="w-full" disabled={load}>
						{load && <Spinner />}
						<Text uuid="login.login_button" />
					</Button>
					<PasskeyLoginButton onSuccess={finishLogin} disabled={load} />
					<p className="text-sm text-muted-foreground text-center">
						<Text uuid="login.no_account" />{" "}
						<a
							href={`${LANDING_URL}/#contact`}
							className="font-medium text-primary hover:underline underline-offset-4"
						>
							<Text uuid="login.contact_us" />
						</a>
					</p>
				</div>
			)}
		</Form>
	);

	function detectCapsLock(e: React.KeyboardEvent<HTMLInputElement>) {
		setCapsLock(e.getModifierState("CapsLock"));
	}

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoad(true);
		setEmailNotVerified(false);
		const response = await userLogin(values);
		setLoad(false);
		if (!response.success) {
			if (response.data?.error_code === "E-AUTH-011") {
				setEmailNotVerified(true);
			}
			return;
		}
		if (isTwoFactorChallenge(response.data)) {
			setMfaToken(response.data.mfa_token);
			return;
		}
		finishLogin(response.data);
	}

	/** Same for a plain login and one that passed the second factor. */
	function finishLogin(session: LoginResponse) {
		setMfaToken(undefined);
		setToken(session.token);
		if (intent === "new_company") {
			navigate("/companies/new");
			return;
		}
		navigate(`/login/environments?exchange_token=${session.exchange_token}`);
	}
};

export default LoginForm;
