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
import { features } from "@/api/feature-service";
import { PermissionPicker } from "@/components/features/permission-picker";
import { Form } from "@/components/forms/form";
import { useText } from "@/hooks/use-text";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";

const formSchema = z.object({
	code: z.string().min(2),
	name: z.string().min(2),
});

const CreateFeature = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { saving, save } = useResourceItem(features);
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
					defaultValues={{ code: "", name: "" }}
				>
					{(field) => (
						<Card>
							<CardHeader>
								<CardTitle>
									{textGet("backoffice.features.create.title")}
								</CardTitle>
								<CardDescription>
									{textGet("backoffice.features.create.description")}
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-4">
								<FormInput
									field={field}
									name="code"
									type="text"
									label={textGet("backoffice.features.col.code")}
									placeholder="CLINICAL, APPOINTMENTS..."
								/>
								<FormInput
									field={field}
									name="name"
									type="text"
									label={textGet("backoffice.features.col.name")}
									placeholder={textGet(
										"backoffice.features.col.name.placeholder",
									)}
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
									onClick={() => navigate("/features")}
								>
									{textGet("backoffice.common.cancel")}
								</Button>
								<Button type="submit" disabled={saving}>
									{saving && <Spinner />}
									{textGet("backoffice.features.create")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</ResourceEditPage>
	);
};

export default CreateFeature;
