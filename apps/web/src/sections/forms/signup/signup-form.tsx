import React from "react";
import { useNavigate, useSearchParams } from "react-router";
import z from "zod";
import { companySignup } from "@/api/auth-service";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { FormPasswordInput } from "@/components/forms/form-input-password";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";

const formSchema = z
	.object({
		name: z.string().min(2),
		user_name: z.string().min(3).regex(/^\S+$/, {
			message: "form.validation.no_spaces",
		}),
		email: z.email(),
		password: z
			.string()
			.min(6)
			.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
		confirm_password: z
			.string()
			.min(6)
			.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
	})
	.refine((data) => data.password === data.confirm_password, {
		message: "form.validation.passwords_no_match",
		path: ["confirm_password"],
	});

const SignupForm = () => {
	const { textGet } = useText();
	const [load, setLoad] = React.useState(false);
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const token = searchParams.get("token");

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

	return (
		<Form<typeof formSchema>
			schema={formSchema}
			onSubmit={onSubmit}
			className="max-w-md w-full min-w-xs"
			defaultValues={{
				name: "",
				user_name: "",
				email: "",
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
								onClick={() => navigate("/login")}
							>
								<Text uuid="signup.go_to_login" />
							</Button>
						</CardFooter>
					</Card>
				);
			}}
		</Form>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!token) return;
		setLoad(true);
		const response = await companySignup({
			token,
			name: values.name,
			user_name: values.user_name,
			email: values.email,
			password: values.password,
		});
		if (response.success) {
			setLoad(false);
			navigate("/login");
			return;
		}
		setLoad(false);
	}
};

export default SignupForm;
