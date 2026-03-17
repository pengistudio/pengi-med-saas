import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import { getPermissions, type Permission } from "@/api/permission-service";
import { createRole } from "@/api/role-service";
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
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import { useText } from "@/hooks/use-text";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const formSchema = z.object({ role: z.string().min(2) });

const CreateRole = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [loading, setLoading] = React.useState(false);
	const [permissions, setPermissions] = React.useState<Permission[]>([]);
	const [selectedPermissions, setSelectedPermissions] = React.useState<
		string[]
	>([]);

	React.useEffect(() => {
		getPermissions().then((res) => {
			if (res.success && res.data) setPermissions(res.data as Permission[]);
		});
	}, []);

	const grouped = React.useMemo(() => {
		const map: Record<string, Permission[]> = {};
		for (const p of permissions) {
			const cat = p.category || "General";
			if (!map[cat]) map[cat] = [];
			map[cat].push(p);
		}
		return Object.entries(map).sort(([a], [b]) => a.localeCompare(b));
	}, [permissions]);

	const togglePermission = (id: string) =>
		setSelectedPermissions((prev) =>
			prev.includes(id) ? prev.filter((p) => p !== id) : [...prev, id],
		);

	const selectAll = () => setSelectedPermissions(permissions.map((p) => p.ID));
	const deselectAll = () => setSelectedPermissions([]);
	const allSelected =
		permissions.length > 0 && selectedPermissions.length === permissions.length;

	const selectAllCategory = (perms: Permission[]) =>
		setSelectedPermissions((prev) =>
			Array.from(new Set([...prev, ...perms.map((p) => p.ID)])),
		);
	const deselectAllCategory = (perms: Permission[]) => {
		const ids = new Set(perms.map((p) => p.ID));
		setSelectedPermissions((prev) => prev.filter((id) => !ids.has(id)));
	};
	const allCategorySelected = (perms: Permission[]) =>
		perms.every((p) => selectedPermissions.includes(p.ID));

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoading(true);
		const res = await createRole({
			...values,
			permission_ids: selectedPermissions,
		});
		setLoading(false);
		if (res.success) navigate("/roles");
	}

	return (
		<DashboardLayout>
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
									onClick={() => navigate("/roles")}
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

export default CreateRole;
