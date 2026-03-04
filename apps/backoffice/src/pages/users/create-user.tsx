import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import { createUser } from "@/api/user-service";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
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
import { DashboardLayout } from "@/sections/template/dashboard-template";

const formSchema = z.object({
	name: z.string().min(2),
	user_name: z.string().min(2),
	password: z.string().min(6),
});

const CreateUser = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [loading, setLoading] = React.useState(false);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoading(true);
		const res = await createUser(values);
		setLoading(false);
		if (res.success) navigate("/users");
	}

	return (
		<DashboardLayout>
			<div className="max-w-2xl mx-auto">
				<Form<typeof formSchema>
					schema={formSchema}
					onSubmit={onSubmit}
					defaultValues={{ name: "", user_name: "", password: "" }}
				>
					{(field) => (
						<Card>
							<CardHeader>
								<CardTitle>
									{textGet("backoffice.users.create.title")}
								</CardTitle>
								<CardDescription>
									{textGet("backoffice.users.create.description")}
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-4">
								<FormInput
									field={field}
									name="name"
									type="text"
									label={textGet("backoffice.users.col.name")}
									placeholder={textGet("backoffice.users.col.name.placeholder")}
								/>
								<FormInput
									field={field}
									name="user_name"
									type="text"
									label={textGet("backoffice.users.col.user_name")}
									placeholder={textGet(
										"backoffice.users.col.user_name.placeholder",
									)}
								/>
								<FormInput
									field={field}
									name="password"
									type="password"
									label={textGet("backoffice.users.col.password")}
									placeholder="••••••••"
								/>
							</CardContent>
							<CardFooter className="flex justify-between">
								<Button
									type="button"
									variant="outline"
									onClick={() => navigate("/users")}
								>
									{textGet("backoffice.companies.cancel")}
								</Button>
								<Button type="submit" disabled={loading}>
									{loading && <Spinner />}
									{textGet("backoffice.users.create")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</DashboardLayout>
	);
};

export default CreateUser;
