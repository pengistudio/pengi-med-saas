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
import { features } from "@/api/feature-service";
import { PermissionPicker } from "@/components/features/permission-picker";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";

const formSchema = z.object({ name: z.string().min(2) });

const EditFeature = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const {
		item: feature,
		loading,
		saving,
		save,
	} = useResourceItem(features, id);
	const code = feature?.code ?? "";
	const defaultValues = { name: feature?.name ?? "" };
	const [selectedPermissions, setSelectedPermissions] = React.useState<
		string[]
	>([]);

	// Load the saved selection once the feature arrives (adjust state during render).
	const [prevFeature, setPrevFeature] = React.useState<typeof feature>();
	if (feature !== prevFeature) {
		setPrevFeature(feature);
		if (feature)
			setSelectedPermissions(feature.permissions?.map((p) => p.ID) ?? []);
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
								<CardTitle>
									{textGet("backoffice.features.edit.title")}
								</CardTitle>
								<CardDescription>
									{textGet("backoffice.features.edit.description")}
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-4">
								<div>
									<span className="text-sm font-medium">
										{textGet("backoffice.features.col.code")}
									</span>
									<p className="mt-1 text-sm font-mono text-muted-foreground bg-muted px-3 py-2 rounded-md">
										{code}
									</p>
								</div>
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

export default EditFeature;
