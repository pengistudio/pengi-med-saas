import { Pencil, Plus, Trash2 } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import { deleteRole, getRoles, type Role } from "@/api/role-service";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { useText } from "@/hooks/use-text";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const RoleList = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [roles, setRoles] = React.useState<Role[]>([]);
	const [loading, setLoading] = React.useState(true);

	const fetch = React.useCallback(async () => {
		setLoading(true);
		const res = await getRoles();
		if (res.success && res.data) setRoles(res.data as Role[]);
		setLoading(false);
	}, []);

	React.useEffect(() => {
		fetch();
	}, [fetch]);

	const handleDelete = async (id: number) => {
		const res = await deleteRole(id);
		if (res.success) fetch();
	};

	return (
		<DashboardLayout>
			<div className="space-y-6">
				<div className="flex items-center justify-between">
					<h1 className="text-2xl font-bold tracking-tight">
						{textGet("backoffice.roles.title")}
					</h1>
					<Button onClick={() => navigate("/roles/create")}>
						<Plus className="h-4 w-4 mr-2" />
						{textGet("backoffice.roles.create")}
					</Button>
				</div>
				<Card>
					<CardHeader>
						<CardTitle>{textGet("backoffice.roles.list.title")}</CardTitle>
						<CardDescription>
							{textGet("backoffice.roles.list.description")}
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loading ? (
							<p className="text-sm text-muted-foreground py-8 text-center animate-pulse">
								{textGet("backoffice.companies.loading")}
							</p>
						) : roles.length === 0 ? (
							<p className="text-sm text-muted-foreground py-8 text-center">
								{textGet("backoffice.roles.empty")}
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>
											{textGet("backoffice.roles.col.name")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.features.col.permissions")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("table.actions")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{roles.map((r) => (
										<TableRow key={r.ID}>
											<TableCell className="font-medium">{r.role}</TableCell>
											<TableCell>
												<span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
													{r.permissions?.length ?? 0}
												</span>
											</TableCell>
											<TableCell className="text-right">
												<div className="flex items-center justify-end gap-2">
													<Button
														variant="ghost"
														size="icon"
														onClick={() => navigate(`/roles/edit/${r.ID}`)}
													>
														<Pencil className="h-4 w-4" />
													</Button>
													<Button
														variant="ghost"
														size="icon"
														onClick={() => handleDelete(r.ID)}
													>
														<Trash2 className="h-4 w-4 text-destructive" />
													</Button>
												</div>
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</CardContent>
				</Card>
			</div>
		</DashboardLayout>
	);
};

export default RoleList;
