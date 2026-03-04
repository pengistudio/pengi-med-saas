import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import { userLogin } from "@/api/auth-service";
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
import { useText } from "@/hooks/use-text";
import { useTokenStore } from "@/store/token-store";

const formSchema = z.object({
	password: z.string().min(6),
	user_name: z.string().min(3),
});

const LoginForm = () => {
	const [load, setLoad] = React.useState(false);
	const { setToken } = useTokenStore();
	const { textGet } = useText();
	const navigate = useNavigate();

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
						</CardFooter>
					</Card>
				);
			}}
		</Form>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoad(true);
		const response = await userLogin(values);
		if (!response.success) {
			setLoad(false);
			return;
		}
		setToken(response.data.token);
		setLoad(false);
		navigate("/");
	}
};

export default LoginForm;
