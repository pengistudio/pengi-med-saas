import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	Checkbox,
	FormInput,
	Label,
	Spinner,
} from "@pengi/ui";
import React from "react";
import { useNavigate, useParams } from "react-router";
import z from "zod";
import { features } from "@/api/feature-service";
import { getPermissions, type Permission } from "@/api/permission-service";
import { Form } from "@/components/forms/form";
import { useText } from "@/hooks/use-text";
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
	const [permissions, setPermissions] = React.useState<Permission[]>([]);
	const [selectedPermissions, setSelectedPermissions] = React.useState<
		string[]
	>([]);

	React.useEffect(() => {
		getPermissions().then((res) => {
			if (res.success && res.data) setPermissions(res.data as Permission[]);
		});
	}, []);

	React.useEffect(() => {
		if (feature)
			setSelectedPermissions(feature.permissions?.map((p) => p.ID) ?? []);
	}, [feature]);

	const grouped = React.useMemo(() => {
		const map: Record<string, Permission[]> = {};
		for (const p of permissions) {
			const cat = p.category || "General";
			if (!map[cat]) map[cat] = [];
			map[cat].push(p);
		}
		return Object.entries(map).sort(([a], [b]) => a.localeCompare(b));
	}, [permissions]);

	const togglePermission = (permId: string) =>
		setSelectedPermissions((prev) =>
			prev.includes(permId)
				? prev.filter((p) => p !== permId)
				: [...prev, permId],
		);

	const selectAll = () => setSelectedPermissions(permissions.map((p) => p.ID));
	const deselectAll = () => setSelectedPermissions([]);
	const allSelected =
		permissions.length > 0 && selectedPermissions.length === permissions.length;

	const selectAllCategory = (perms: Permission[]) => {
		setSelectedPermissions((prev) => {
			const ids = new Set(prev);
			for (const p of perms) ids.add(p.ID);
			return Array.from(ids);
		});
	};
	const deselectAllCategory = (perms: Permission[]) => {
		const idsToRemove = new Set(perms.map((p) => p.ID));
		setSelectedPermissions((prev) =>
			prev.filter((pid) => !idsToRemove.has(pid)),
		);
	};
	const allCategorySelected = (perms: Permission[]) =>
		perms.every((p) => selectedPermissions.includes(p.ID));

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

								{permissions.length > 0 && (
									<div className="space-y-3">
										<div className="flex items-center justify-between">
											<Label>
												{textGet("backoffice.features.col.permissions")}
											</Label>
											<Button
												type="button"
												variant="ghost"
												size="sm"
												className="text-xs h-7"
												onClick={allSelected ? deselectAll : selectAll}
											>
												{allSelected
													? textGet("backoffice.linking.deselect_all")
													: textGet("backoffice.linking.select_all")}
											</Button>
										</div>
										<div className="max-h-72 overflow-y-auto border rounded-md p-3 space-y-4">
											{grouped.map(([category, perms]) => (
												<div key={category}>
													<div className="flex items-center justify-between mb-2 pb-1 border-b">
														<span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
															{category}
														</span>
														<Button
															type="button"
															variant="ghost"
															size="sm"
															className="text-xs h-6 px-2"
															onClick={() =>
																allCategorySelected(perms)
																	? deselectAllCategory(perms)
																	: selectAllCategory(perms)
															}
														>
															{allCategorySelected(perms)
																? textGet("backoffice.linking.deselect_all")
																: textGet("backoffice.linking.select_all")}
														</Button>
													</div>
													<div className="grid grid-cols-1 gap-1">
														{perms.map((p) => (
															<span
																key={p.ID}
																className="flex items-center gap-2 cursor-pointer hover:bg-muted/50 rounded px-2 py-1.5 transition-colors"
															>
																<Checkbox
																	checked={selectedPermissions.includes(p.ID)}
																	onCheckedChange={() => togglePermission(p.ID)}
																/>
																<div className="flex flex-col">
																	<span className="text-sm font-medium">
																		{p.name}
																	</span>
																	<span className="text-xs text-muted-foreground">
																		{p.ID}
																	</span>
																</div>
															</span>
														))}
													</div>
												</div>
											))}
										</div>
										<p className="text-xs text-muted-foreground">
											{selectedPermissions.length} / {permissions.length}{" "}
											{textGet("backoffice.linking.selected")}
										</p>
									</div>
								)}
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
