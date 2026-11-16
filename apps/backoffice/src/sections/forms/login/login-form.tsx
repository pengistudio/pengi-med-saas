import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	Form,
	FormInput,
	FormPasswordInput,
	Spinner,
} from "@pengi/ui";
import React from "react";
import z from "zod";
import {
	initialStep,
	type LoginEvent,
	type LoginOutcome,
	nextStep,
	useSession,
} from "@/lib/session";
import { EnrollmentCodesStep, EnrollmentStep } from "./enrollment-step";
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

/**
 * The sign-in: password -> second factor (code or passkey) or, for an admin
 * with no second factor, forced enrollment. Every token lives in this state
 * only; the session starts when the last step finishes.
 */
const LoginForm = () => {
	const { establish } = useSession();
	const [state, setState] = React.useState(initialStep);
	const dispatch = (event: LoginEvent) =>
		setState((current) => nextStep(current, event));

	switch (state.step) {
		case "second_factor":
			return (
				<TwoFactorStep
					mfaToken={state.mfaToken}
					totp={state.totp}
					passkey={state.passkey}
					onSuccess={establish}
					onBack={() => dispatch({ type: "back" })}
					onExpired={() => dispatch({ type: "expired" })}
				/>
			);
		case "enroll":
			return (
				<EnrollmentStep
					enrollmentToken={state.enrollmentToken}
					passkeysAvailable={state.passkeysAvailable}
					onSignedIn={establish}
					onTotpEnrolled={(token, codes) =>
						dispatch({ type: "totp_enrolled", token, codes })
					}
					onBack={() => dispatch({ type: "back" })}
					onExpired={() => dispatch({ type: "expired" })}
				/>
			);
		case "recovery_codes":
			return (
				<EnrollmentCodesStep
					codes={state.codes}
					onDone={() => establish(state.token)}
				/>
			);
		default:
			return (
				<PasswordStep
					onOutcome={(outcome) => dispatch({ type: "password", outcome })}
					onPasskey={establish}
				/>
			);
	}
};

const PasswordStep = ({
	onOutcome,
	onPasskey,
}: {
	onOutcome: (outcome: LoginOutcome) => void;
	onPasskey: (token: string) => void;
}) => {
	const [load, setLoad] = React.useState(false);
	const { login } = useSession();
	const { textGet } = useText();

	return (
		<Form<typeof formSchema>
			schema={formSchema}
			onSubmit={onSubmit}
			className="max-w-md w-full min-w-xs"
			defaultValues={{
				password: "",
				user_name: "",
			}}
		>
			{(field) => {
				return (
					<Card className="md:shadow-none md:border-none">
						<CardHeader>
							<CardTitle>{textGet("backoffice.login.title")}</CardTitle>
							<CardDescription>
								{textGet("backoffice.login.subtitle")}
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-4">
							<FormInput
								field={field}
								name="user_name"
								type="text"
								placeholder={textGet("backoffice.login.username.placeholder")}
								label={textGet("backoffice.login.username")}
								autoComplete="username"
							/>
							<FormPasswordInput
								field={field}
								name="password"
								placeholder={textGet("backoffice.login.password.placeholder")}
								label={textGet("backoffice.login.password")}
							/>
						</CardContent>
						<CardFooter className="flex flex-col gap-4">
							<Button type="submit" className="w-full" disabled={load}>
								{load && <Spinner />}
								{textGet("backoffice.login.button")}
							</Button>
							<PasskeyLoginButton onSuccess={onPasskey} disabled={load} />
						</CardFooter>
					</Card>
				);
			}}
		</Form>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoad(true);
		const outcome = await login(values);
		setLoad(false);
		onOutcome(outcome);
	}
};

export default LoginForm;
