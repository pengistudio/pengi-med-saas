import React from "react";
import { useNavigate, useParams } from "react-router";
import z from "zod";
import { getUserByID, updateUser } from "@/api/user-service";
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
	password: z.string().optional(),
});

const EditUser = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const [loading, setLoading] = React.useState(false);
	const [initialLoading, setInitialLoading] = React.useState(true);
	const [defaultValues, setDefaultValues] = React.useState({
		name: "",
		user_name: "",
		password: "",
	});

	React.useEffect(() => {
		if (!id) return;
		getUserByID(id).then((res) => {
			if (res.success && res.data) {
				setDefaultValues({
					name: res.data.name,
					user_name: res.data.user_name,
					password: "",
				});
			}
			setInitialLoading(false);
		});
	}, [id]);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!id) return;
		setLoading(true);
		const payload: Record<string, string> = {};
		if (values.name) payload.name = values.name;
		if (values.user_name) payload.user_name = values.user_name;
		if (values.password) payload.password = values.password;
		const res = await updateUser(id, payload);
		setLoading(false);
		if (res.success) navigate("/users");
	}

	if (initialLoading) {
		return (
			<DashboardLayout>
				<div className="flex items-center justify-center h-64">
					<p className="text-muted-foreground animate-pulse">
						{textGet("backoffice.companies.loading")}
					</p>
				</div>
			</DashboardLayout>
		);
	}

	return (
		<DashboardLayout>
			<div className="max-w-2xl mx-auto">
				<Form<typeof formSchema>
					schema={formSchema}
					onSubmit={onSubmit}
					defaultValues={defaultValues}
				>
					{(field) => (
						<Card>
							<CardHeader>
								<CardTitle>{textGet("backoffice.users.edit.title")}</CardTitle>
								<CardDescription>
									{textGet("backoffice.users.edit.description")}
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
									label={textGet("backoffice.users.col.password_new")}
									placeholder={textGet(
										"backoffice.users.col.password_new.placeholder",
									)}
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
									{textGet("backoffice.companies.save")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</DashboardLayout>
	);
};

export default EditUser;
