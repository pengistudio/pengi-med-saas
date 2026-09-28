import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	Form,
	FormInput,
	Spinner,
} from "@pengi/ui";
import React from "react";
import { useNavigate, useParams } from "react-router";
import z from "zod";
import { roles } from "@/api/role-service";
import { PermissionPicker } from "@/components/features/permission-picker";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";

const formSchema = z.object({ role: z.string().min(2) });

const EditRole = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const { item: role, loading, saving, save } = useResourceItem(roles, id);
	const defaultValues = { role: role?.role ?? "" };
	const [selectedPermissions, setSelectedPermissions] = React.useState<
		string[]
	>([]);

	// Load the saved selection once the role arrives (adjust state during render).
	const [prevRole, setPrevRole] = React.useState<typeof role>();
	if (role !== prevRole) {
		setPrevRole(role);
		if (role) setSelectedPermissions(role.permissions?.map((p) => p.ID) ?? []);
	}

	async function onSubmit(values: z.infer<typeof formSchema>) {
		await save({
			...values,
			permission_ids: selectedPermissions,
		});
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
								<CardTitle>{textGet("backoffice.roles.edit.title")}</CardTitle>
								<CardDescription>
									{textGet("backoffice.roles.edit.description")}
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

export default EditRole;
