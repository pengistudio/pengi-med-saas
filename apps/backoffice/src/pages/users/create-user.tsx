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
import { useNavigate } from "react-router";
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
		.min(6)
		.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
});

const CreateUser = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { saving, save } = useResourceItem(users);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		await save(values);
	}

	return (
		<ResourceEditPage>
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
									{textGet("backoffice.common.cancel")}
								</Button>
								<Button type="submit" disabled={saving}>
									{saving && <Spinner />}
									{textGet("backoffice.users.create")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</ResourceEditPage>
	);
};

export default CreateUser;
