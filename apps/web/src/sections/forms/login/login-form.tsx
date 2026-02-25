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
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";
import { useTokenStore } from "@/store/token-store";

const formSchema = z.object({
	password: z.string().min(6),
	user_name: z.string().min(3),
});

const LoginForm = () => {
	const { textGet } = useText();
	const [load, setLoad] = React.useState(false);
	const { setToken } = useTokenStore();
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
							<CardTitle>
								<Text uuid="login.title" />
							</CardTitle>
							<CardDescription>
								<Text uuid="login.subtitle" />
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-4">
							<FormInput
								field={field}
								name="user_name"
								type="text"
								placeholder={textGet("login.username.placeholder")}
								label={textGet("login.username")}
								autoComplete="username"
							/>
							<FormPasswordInput
								field={field}
								name="password"
								placeholder={textGet("login.password.placeholder")}
								label={textGet("login.password")}
							/>
						</CardContent>
						<CardFooter className="flex flex-col gap-4">
							<Button type="submit" className="w-full" disabled={load}>
								{load && <Spinner />}
								<Text uuid="login.login_button" />
							</Button>
							<Separator className="w-full" />
							<Button className="w-full" variant="outline" type="button">
								<Text uuid="session.register.button" />
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
		navigate(
			`/login/environments?exchange_token=${response.data.exchange_token}`,
		);
	}
};

export default LoginForm;
