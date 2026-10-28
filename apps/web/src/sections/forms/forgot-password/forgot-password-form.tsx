import { useText } from "@pengi/shared";
import { Button, Form, FormInput, Spinner, Text } from "@pengi/ui";
import React from "react";
import { Link } from "react-router";
import z from "zod";
import { forgotPassword } from "@/api/auth-service";
import { SUPPORT } from "@/lib/constants";
import { generateWhatsAppLink } from "@/lib/utils";

const formSchema = z.object({
	identifier: z.string().trim().min(3),
});

const linkClass = "font-medium text-primary hover:underline underline-offset-4";

const ForgotPasswordForm = () => {
	const { textGet } = useText();
	const [loading, setLoading] = React.useState(false);
	// The identifier the link was sent for; set once the request succeeds.
	const [sentTo, setSentTo] = React.useState<string>();

	if (sentTo) {
		return (
			<div className="space-y-6">
				<div className="space-y-2">
					<h1 className="text-3xl font-semibold tracking-tight">
						<Text uuid="forgot_password.sent.title" />
					</h1>
					<p className="text-muted-foreground">
						{textGet("forgot_password.sent.description", {
							identifier: sentTo,
						})}
					</p>
				</div>
				<Link to="/login" className={`text-sm ${linkClass}`}>
					<Text uuid="forgot_password.back_to_login" />
				</Link>
			</div>
		);
	}

	return (
		<Form<typeof formSchema>
			schema={formSchema}
			onSubmit={onSubmit}
			className="w-full"
			defaultValues={{ identifier: "" }}
		>
			{(field) => (
				<div className="space-y-6">
					<div className="space-y-2">
						<h1 className="text-3xl font-semibold tracking-tight">
							<Text uuid="forgot_password.title" />
						</h1>
						<p className="text-muted-foreground">
							<Text uuid="forgot_password.description" />
						</p>
					</div>
					<FormInput
						field={field}
						name="identifier"
						type="text"
						label={textGet("forgot_password.identifier")}
						autoComplete="username"
						autoFocus
					/>
					<Button type="submit" size="lg" className="w-full" disabled={loading}>
						{loading && <Spinner />}
						<Text uuid="forgot_password.submit" />
					</Button>
					<div className="flex flex-col items-center gap-2 text-sm">
						<Link to="/login" className={linkClass}>
							<Text uuid="forgot_password.back_to_login" />
						</Link>
						<a
							href={generateWhatsAppLink(
								SUPPORT.WHATSAPP,
								textGet("forgot_password.no_email.message"),
							)}
							target="_blank"
							rel="noreferrer"
							className="text-muted-foreground hover:text-foreground hover:underline underline-offset-4"
						>
							<Text uuid="forgot_password.no_email" />
						</a>
					</div>
				</div>
			)}
		</Form>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoading(true);
		const res = await forgotPassword(values.identifier);
		setLoading(false);
		if (res.success) {
			setSentTo(values.identifier);
		}
	}
};

export default ForgotPasswordForm;
