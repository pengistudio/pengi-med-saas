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
import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import { roles } from "@/api/role-service";
import { PermissionPicker } from "@/components/features/permission-picker";
import { Form } from "@/components/forms/form";
import { useText } from "@/hooks/use-text";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";

const formSchema = z.object({ role: z.string().min(2) });

const CreateRole = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { saving, save } = useResourceItem(roles);
	const [selectedPermissions, setSelectedPermissions] = React.useState<
		string[]
	>([]);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		await save({
			...values,
			permission_ids: selectedPermissions,
		});
	}

	return (
		<ResourceEditPage>
			<div className="max-w-2xl mx-auto">
				<Form<typeof formSchema>
					schema={formSchema}
					onSubmit={onSubmit}
					defaultValues={{ role: "" }}
				>
					{(field) => (
						<Card>
							<CardHeader>
								<CardTitle>
									{textGet("backoffice.roles.create.title")}
								</CardTitle>
								<CardDescription>
									{textGet("backoffice.roles.create.description")}
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-4">
								<FormInput
									field={field}
									name="role"
									type="text"
									label={textGet("backoffice.roles.col.name")}
									placeholder={textGet("backoffice.roles.col.name.placeholder")}
								/>

								<PermissionPicker
									value={selectedPermissions}
									onChange={setSelectedPermissions}
								/>
							</CardContent>
							<CardFooter className="flex justify-between">
								<Button
									type="button"
									variant="outline"
									onClick={() => navigate("/roles")}
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

export default CreateRole;
