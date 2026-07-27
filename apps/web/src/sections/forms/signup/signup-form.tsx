import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	FormInput,
	FormPasswordInput,
	Spinner,
	Text,
} from "@pengi/ui";
import React from "react";
import { useNavigate, useSearchParams } from "react-router";
import z from "zod";
import {
	checkCompanySignupEmail,
	companySignup,
	joinCompanyWithExistingAccount,
} from "@/api/auth-service";
import { Form } from "@/components/forms/form";
import { useText } from "@/hooks/use-text";
import { useTokenStore } from "@/store/token-store";

const emailStepSchema = z.object({
	email: z.email(),
});

const signupSchema = z
	.object({
		name: z.string().min(2).max(100),
		user_name: z
			.string()
			.min(3)
			.max(50)
			.regex(/^[a-zA-Z0-9_.-]+$/, {
				message: "form.validation.username_invalid",
			}),
		email: z.email(),
		password: z
			.string()
			.min(6)
			.regex(/^\S+$/, { message: "form.validation.no_spaces" })
			.regex(/[A-Z]/, { message: "form.validation.password_uppercase" })
			.regex(/[0-9]/, { message: "form.validation.password_number" }),
		confirm_password: z
			.string()
			.min(6)
			.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
	})
	.refine((data) => data.password === data.confirm_password, {
		message: "form.validation.passwords_no_match",
		path: ["confirm_password"],
	});

const existingAccountSchema = z.object({
	password: z
		.string()
		.min(6)
		.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
});

type Step = "email" | "new_account" | "existing_account";

const SignupForm = () => {
	const { textGet } = useText();
	const { setToken } = useTokenStore();
	const [load, setLoad] = React.useState(false);
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const token = searchParams.get("token");
	const honeypotRef = React.useRef<HTMLInputElement>(null);

	const [step, setStep] = React.useState<Step>("email");
	const [knownEmail, setKnownEmail] = React.useState("");

	if (!token) {
		return (
			<Card className="max-w-md w-full min-w-xs">
				<CardHeader>
					<CardTitle>
						<Text uuid="signup.invalid_link.title" />
					</CardTitle>
					<CardDescription>
						<Text uuid="signup.invalid_link.description" />
					</CardDescription>
				</CardHeader>
				<CardFooter className="flex flex-col gap-4">
					<Button className="w-full" onClick={() => navigate("/login")}>
						<Text uuid="signup.go_to_login" />
					</Button>
				</CardFooter>
			</Card>
		);
	}

	async function onEmailStepSubmit(values: z.infer<typeof emailStepSchema>) {
		if (!token) return;
		setLoad(true);
		const res = await checkCompanySignupEmail(token, values.email);
		setLoad(false);
		if (!res.success) return;
		setKnownEmail(values.email);
		setStep(res.data?.exists ? "existing_account" : "new_account");
	}

	async function onSignupSubmit(values: z.infer<typeof signupSchema>) {
		if (!token) return;
		if (honeypotRef.current?.value) {
			navigate("/login");
			return;
		}
		setLoad(true);
		const response = await companySignup({
			token,
			name: values.name,
			user_name: values.user_name,
			email: values.email,
			password: values.password,
		});
		setLoad(false);
		if (response.success) {
			navigate("/login");
		}
	}

	async function onExistingAccountSubmit(
		values: z.infer<typeof existingAccountSchema>,
	) {
		if (!token) return;
		setLoad(true);
		const response = await joinCompanyWithExistingAccount({
			token,
			email: knownEmail,
			password: values.password,
		});
		setLoad(false);
		if (response.success && response.data) {
			setToken(response.data.token);
			navigate(
				`/login/environments?exchange_token=${response.data.exchange_token}`,
			);
		}
	}

	if (step === "email") {
		return (
			<Form<typeof emailStepSchema>
				schema={emailStepSchema}
				onSubmit={onEmailStepSubmit}
				className="max-w-md w-full min-w-xs"
				defaultValues={{ email: "" }}
			>
				{(field) => (
					<Card className="md:shadow-none md:border-none">
						<CardHeader>
							<CardTitle>
								<Text uuid="signup.title" />
							</CardTitle>
							<CardDescription>
								<Text uuid="signup.subtitle" />
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-4">
							<FormInput
								field={field}
								name="email"
								type="email"
								placeholder={textGet("signup.email.placeholder")}
								label={textGet("signup.email")}
								autoComplete="email"
							/>
						</CardContent>
						<CardFooter className="flex flex-col gap-4">
							<Button type="submit" className="w-full" disabled={load}>
								{load && <Spinner />}
								<Text uuid="signup.email_step.continue" />
							</Button>
							<Button
								variant="link"
								className="w-full"
								type="button"
								onClick={() => navigate("/login")}
							>
								<Text uuid="signup.go_to_login" />
							</Button>
						</CardFooter>
					</Card>
				)}
			</Form>
		);
	}

	if (step === "existing_account") {
		return (
			<Form<typeof existingAccountSchema>
				schema={existingAccountSchema}
				onSubmit={onExistingAccountSubmit}
				className="max-w-md w-full min-w-xs"
				defaultValues={{ password: "" }}
			>
				{(field) => (
					<Card className="md:shadow-none md:border-none">
						<CardHeader>
							<CardTitle>
								<Text uuid="signup.existing_account.title" />
							</CardTitle>
							<CardDescription>
								{knownEmail} —{" "}
								<Text uuid="signup.existing_account.description" />
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-4">
							<FormPasswordInput
								field={field}
								name="password"
								placeholder={textGet("signup.password.placeholder")}
								label={textGet("signup.password")}
							/>
						</CardContent>
						<CardFooter className="flex flex-col gap-4">
							<Button type="submit" className="w-full" disabled={load}>
								{load && <Spinner />}
								<Text uuid="signup.existing_account.submit" />
							</Button>
							<Button
								variant="link"
								className="w-full"
								type="button"
								onClick={() => setStep("email")}
							>
								<Text uuid="signup.go_to_login" />
							</Button>
						</CardFooter>
					</Card>
				)}
			</Form>
		);
	}

	return (
		<Form<typeof signupSchema>
			schema={signupSchema}
			onSubmit={onSignupSubmit}
			className="max-w-md w-full min-w-xs"
			defaultValues={{
				name: "",
				user_name: "",
				email: knownEmail,
				password: "",
				confirm_password: "",
			}}
		>
			{(field) => {
				return (
					<Card className="md:shadow-none md:border-none">
						<CardHeader>
							<CardTitle>
								<Text uuid="signup.title" />
							</CardTitle>
							<CardDescription>
								<Text uuid="signup.subtitle" />
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-4">
							<FormInput
								field={field}
								name="name"
								type="text"
								placeholder={textGet("signup.name.placeholder")}
								label={textGet("signup.name")}
								autoComplete="name"
							/>
							<FormInput
								field={field}
								name="user_name"
								type="text"
								placeholder={textGet("signup.username.placeholder")}
								label={textGet("signup.username")}
								autoComplete="username"
							/>
							<FormInput
								field={field}
								name="email"
								type="email"
								placeholder={textGet("signup.email.placeholder")}
								label={textGet("signup.email")}
								autoComplete="email"
								readOnly
							/>
							<FormPasswordInput
								field={field}
								name="password"
								placeholder={textGet("signup.password.placeholder")}
								label={textGet("signup.password")}
							/>
							<FormPasswordInput
								field={field}
								name="confirm_password"
								placeholder={textGet("signup.confirm_password.placeholder")}
								label={textGet("signup.confirm_password")}
							/>
							{/* honeypot: bots fill this, humans don't see it */}
							<input
								ref={honeypotRef}
								type="text"
								name="website"
								autoComplete="off"
								tabIndex={-1}
								aria-hidden="true"
								style={{
									position: "absolute",
									left: "-9999px",
									opacity: 0,
									height: 0,
									width: 0,
									overflow: "hidden",
								}}
							/>
						</CardContent>
						<CardFooter className="flex flex-col gap-4">
							<Button type="submit" className="w-full" disabled={load}>
								{load && <Spinner />}
								<Text uuid="signup.submit_button" />
							</Button>
							<Button
								variant="link"
								className="w-full"
								type="button"
								onClick={() => setStep("email")}
							>
								<Text uuid="signup.go_to_login" />
							</Button>
						</CardFooter>
					</Card>
				);
			}}
		</Form>
	);
};

export default SignupForm;
