import React from "react";
import { useNavigate, useSearchParams } from "react-router";
import z from "zod";
import { resetPassword } from "@/api/auth-service";
import { Form } from "@/components/forms/form";
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
		new_password: z.string().min(6),
		confirm_password: z.string().min(6),
	})
	.refine((data) => data.new_password === data.confirm_password, {
		message: "Las contraseñas no coinciden",
		path: ["confirm_password"],
	});

const ResetPasswordForm = () => {
	const { textGet } = useText();
	const [loading, setLoading] = React.useState(false);
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const token = searchParams.get("token");

	if (!token) {
		return (
			<Card className="max-w-md w-full min-w-xs">
				<CardHeader>
					<CardTitle>
						<Text uuid="reset_password.invalid_link.title" />
					</CardTitle>
					<CardDescription>
						<Text uuid="reset_password.invalid_link.description" />
					</CardDescription>
				</CardHeader>
				<CardFooter>
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
			defaultValues={{ new_password: "", confirm_password: "" }}
		>
			{(field) => (
				<Card className="md:shadow-none md:border-none">
					<CardHeader>
						<CardTitle>
							<Text uuid="reset_password.title" />
						</CardTitle>
						<CardDescription>
							<Text uuid="reset_password.description" />
						</CardDescription>
					</CardHeader>
					<CardContent className="space-y-4">
						<FormPasswordInput
							field={field}
							name="new_password"
							label={textGet("reset_password.new_password")}
							placeholder={textGet("reset_password.new_password.placeholder")}
						/>
						<FormPasswordInput
							field={field}
							name="confirm_password"
							label={textGet("reset_password.confirm_password")}
							placeholder={textGet(
								"reset_password.confirm_password.placeholder",
							)}
						/>
					</CardContent>
					<CardFooter className="flex flex-col gap-4">
						<Button type="submit" className="w-full" disabled={loading}>
							{loading && <Spinner />}
							<Text uuid="reset_password.submit" />
						</Button>
						<Button
							type="button"
							variant="link"
							className="w-full"
							onClick={() => navigate("/login")}
						>
							<Text uuid="signup.go_to_login" />
						</Button>
					</CardFooter>
				</Card>
			)}
		</Form>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!token) return;
		setLoading(true);
		const res = await resetPassword(token, values.new_password);
		setLoading(false);
		if (res.success) {
			navigate("/login");
		}
	}
};

export default ResetPasswordForm;
