import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Form,
	FormPasswordInput,
	Text,
} from "@pengi/ui";
import { KeyRound } from "lucide-react";
import React from "react";
import z from "zod";
import { changePassword } from "@/api/auth-service";
import useAuth from "@/hooks/use-auth";

const schema = z
	.object({
		current_password: z.string().min(1),
		new_password: z
			.string()
			.min(10)
			.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
		confirm_password: z.string().min(1),
	})
	.refine((d) => d.new_password === d.confirm_password, {
		message: "form.validation.passwords_no_match",
		path: ["confirm_password"],
	});

/** Change the signed-in user's password; every session is revoked, so sign in again. */
export function ChangePasswordCard() {
	const { textGet } = useText();
	const { logout } = useAuth();
	const [loading, setLoading] = React.useState(false);

	async function onSubmit(values: z.infer<typeof schema>) {
		setLoading(true);
		const res = await changePassword(
			values.current_password,
			values.new_password,
		);
		setLoading(false);
		if (res.success) logout();
	}

	return (
		<Card>
			<CardHeader>
				<CardTitle>
					<Text uuid="profile.security.title" />
				</CardTitle>
				<CardDescription>
					<Text uuid="profile.security.description" />
				</CardDescription>
			</CardHeader>
			<CardContent>
				<Form<typeof schema>
					schema={schema}
					onSubmit={onSubmit}
					className="max-w-md"
					defaultValues={{
						current_password: "",
						new_password: "",
						confirm_password: "",
					}}
				>
					{(field) => (
						<div className="space-y-4">
							<FormPasswordInput
								field={field}
								name="current_password"
								label={textGet("profile.security.current_password")}
							/>
							<FormPasswordInput
								field={field}
								name="new_password"
								label={textGet("profile.security.new_password")}
							/>
							<FormPasswordInput
								field={field}
								name="confirm_password"
								label={textGet("profile.security.confirm_password")}
							/>
							<div className="flex justify-end">
								<Button type="submit" disabled={loading}>
									<KeyRound className="mr-2 h-4 w-4" />
									{textGet("profile.security.submit")}
								</Button>
							</div>
						</div>
					)}
				</Form>
			</CardContent>
		</Card>
	);
}
