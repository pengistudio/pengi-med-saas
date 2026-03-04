import { Pencil, Plus, Trash2 } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import { type BackofficeUser, deleteUser, getUsers } from "@/api/user-service";
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

const UserList = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [users, setUsers] = React.useState<BackofficeUser[]>([]);
	const [loading, setLoading] = React.useState(true);

	const fetchUsers = React.useCallback(async () => {
		setLoading(true);
		const res = await getUsers();
		if (res.success && res.data) setUsers(res.data as BackofficeUser[]);
		setLoading(false);
	}, []);

	React.useEffect(() => {
		fetchUsers();
	}, [fetchUsers]);

	const handleDelete = async (id: number) => {
		const res = await deleteUser(id);
		if (res.success) fetchUsers();
	};

	return (
		<DashboardLayout>
			<div className="space-y-6">
				<div className="flex items-center justify-between">
					<h1 className="text-2xl font-bold tracking-tight">
						{textGet("backoffice.users.title")}
					</h1>
					<Button onClick={() => navigate("/users/create")}>
						<Plus className="h-4 w-4 mr-2" />
						{textGet("backoffice.users.create")}
					</Button>
				</div>

				<Card>
					<CardHeader>
						<CardTitle>{textGet("backoffice.users.list.title")}</CardTitle>
						<CardDescription>
							{textGet("backoffice.users.list.description")}
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loading ? (
							<p className="text-sm text-muted-foreground py-8 text-center animate-pulse">
								{textGet("backoffice.companies.loading")}
							</p>
						) : users.length === 0 ? (
							<p className="text-sm text-muted-foreground py-8 text-center">
								{textGet("backoffice.users.empty")}
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>
											{textGet("backoffice.users.col.name")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.users.col.user_name")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("table.actions")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{users.map((user) => (
										<TableRow key={user.ID}>
											<TableCell className="font-medium">{user.name}</TableCell>
											<TableCell className="text-muted-foreground">
												{user.user_name}
											</TableCell>
											<TableCell className="text-right">
												<div className="flex items-center justify-end gap-2">
													<Button
														variant="ghost"
														size="icon"
														onClick={() => navigate(`/users/edit/${user.ID}`)}
													>
														<Pencil className="h-4 w-4" />
													</Button>
													<Button
														variant="ghost"
														size="icon"
														onClick={() => handleDelete(user.ID)}
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

export default UserList;
