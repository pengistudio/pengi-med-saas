import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	FormInput,
	Spinner,
} from "@pengi/ui";
import { useNavigate, useParams } from "react-router";
import z from "zod";
import { users } from "@/api/user-service";
import { Form } from "@/components/forms/form";
import { useText } from "@/hooks/use-text";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";

const formSchema = z.object({
	name: z.string().min(2),
	user_name: z.string().min(2),
	password: z
		.string()
		.regex(/^\S+$/, { message: "form.validation.no_spaces" })
		.optional(),
});

const EditUser = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const { item, loading, saving, save } = useResourceItem(users, id);
	const defaultValues = {
		name: item?.name ?? "",
		user_name: item?.user_name ?? "",
		password: "",
	};

	async function onSubmit(values: z.infer<typeof formSchema>) {
		const payload: Record<string, string> = {};
		if (values.name) payload.name = values.name;
		if (values.user_name) payload.user_name = values.user_name;
		if (values.password) payload.password = values.password;
		await save(payload);
	}

	return (
		<ResourceEditPage loading={loading}>
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
									{textGet("backoffice.common.cancel")}
								</Button>
								<Button type="submit" disabled={saving}>
									{saving && <Spinner />}
									{textGet("backoffice.common.save")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</ResourceEditPage>
	);
};

export default EditUser;
