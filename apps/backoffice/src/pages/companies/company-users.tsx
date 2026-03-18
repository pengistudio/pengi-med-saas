import { ArrowLeft, Copy, KeyRound, Pencil } from "lucide-react";
import React from "react";
import { useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import z from "zod";
import {
	type CompanyUser,
	getCompanyByID,
	getCompanyUsers,
	getPasswordResetLink,
	getRoles,
	type PasswordResetLinkResponse,
	type Role,
	type UpdateCompanyUserRequest,
	updateCompanyUser,
} from "@/api/company-service";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { FormSelect } from "@/components/forms/form-select";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
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

const editUserSchema = z.object({
	user_name: z.string().min(2),
	email: z.string().email(),
	role_id: z.string().min(1),
});

const CompanyUsers = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const [users, setUsers] = React.useState<CompanyUser[]>([]);
	const [roles, setRoles] = React.useState<Role[]>([]);
	const [loading, setLoading] = React.useState(true);
	const [companyName, setCompanyName] = React.useState("");

	// Edit dialog state
	const [editDialogOpen, setEditDialogOpen] = React.useState(false);
	const [editingUser, setEditingUser] = React.useState<CompanyUser | null>(
		null,
	);
	const [saving, setSaving] = React.useState(false);

	// Password reset link dialog state
	const [resetLinkDialogOpen, setResetLinkDialogOpen] = React.useState(false);
	const [resetLinkData, setResetLinkData] =
		React.useState<PasswordResetLinkResponse | null>(null);
	const [generatingLink, setGeneratingLink] = React.useState(false);

	const fetchData = React.useCallback(async () => {
		if (!id) return;
		setLoading(true);

		const [usersRes, rolesRes, companyRes] = await Promise.all([
			getCompanyUsers(id),
			getRoles(),
			getCompanyByID(id),
		]);

		if (usersRes.success && usersRes.data) {
			setUsers(usersRes.data as CompanyUser[]);
		}
		if (rolesRes.success && rolesRes.data) {
			setRoles(rolesRes.data as Role[]);
		}
		if (companyRes.success && companyRes.data) {
			setCompanyName(companyRes.data.trade_name);
		}

		setLoading(false);
	}, [id]);

	React.useEffect(() => {
		fetchData();
	}, [fetchData]);

	const handleEdit = (user: CompanyUser) => {
		setEditingUser(user);
		setEditDialogOpen(true);
	};

	const handleGenerateResetLink = async (user: CompanyUser) => {
		if (!id) return;
		setGeneratingLink(true);
		const res = await getPasswordResetLink(id, user.user_id);
		setGeneratingLink(false);
		if (res.success && res.data) {
			setResetLinkData(res.data);
			setResetLinkDialogOpen(true);
		}
	};

	const handleCopyLink = (link: string) => {
		navigator.clipboard.writeText(link);
		toast.success(textGet("backoffice.company_users.password_reset.copied"));
	};

	const handleSave = async (values: z.infer<typeof editUserSchema>) => {
		if (!id || !editingUser) return;
		setSaving(true);

		const payload: UpdateCompanyUserRequest = {};
		if (values.user_name !== editingUser.user_name)
			payload.user_name = values.user_name;
		if (values.email !== editingUser.email) payload.email = values.email;
		const newRoleId = Number(values.role_id);
		if (newRoleId !== editingUser.role_id) payload.role_id = newRoleId;

		const res = await updateCompanyUser(id, editingUser.user_id, payload);
		setSaving(false);

		if (res.success) {
			setEditDialogOpen(false);
			fetchData();
		}
	};

	const roleOptions = roles.map((role) => ({
		value: String(role.ID),
		label: role.role,
	}));

	return (
		<>
			<DashboardLayout>
				<div className="space-y-6">
					<div className="flex items-center justify-between">
						<div className="flex items-center gap-3">
							<Button
								variant="ghost"
								size="icon"
								onClick={() => navigate("/companies")}
							>
								<ArrowLeft className="h-4 w-4" />
							</Button>
							<div>
								<h1 className="text-2xl font-bold tracking-tight">
									{textGet("backoffice.company_users.title")}
								</h1>
								{companyName && (
									<p className="text-sm text-muted-foreground">{companyName}</p>
								)}
							</div>
						</div>
					</div>

					<Card>
						<CardHeader>
							<CardTitle>
								{textGet("backoffice.company_users.list.title")}
							</CardTitle>
							<CardDescription>
								{textGet("backoffice.company_users.list.description")}
							</CardDescription>
						</CardHeader>
						<CardContent>
							{loading ? (
								<p className="text-sm text-muted-foreground py-8 text-center animate-pulse">
									{textGet("backoffice.company_users.loading")}
								</p>
							) : users.length === 0 ? (
								<p className="text-sm text-muted-foreground py-8 text-center">
									{textGet("backoffice.company_users.empty")}
								</p>
							) : (
								<Table>
									<TableHeader>
										<TableRow>
											<TableHead>
												{textGet("backoffice.company_users.col.username")}
											</TableHead>
											<TableHead>
												{textGet("backoffice.company_users.col.email")}
											</TableHead>
											<TableHead>
												{textGet("backoffice.company_users.col.role")}
											</TableHead>
											<TableHead>
												{textGet("backoffice.company_users.col.environment")}
											</TableHead>
											<TableHead className="text-right">
												{textGet("table.actions")}
											</TableHead>
										</TableRow>
									</TableHeader>
									<TableBody>
										{users.map((user) => (
											<TableRow key={user.environment_id}>
												<TableCell className="font-medium">
													{user.user_name}
												</TableCell>
												<TableCell>{user.email}</TableCell>
												<TableCell>
													<span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
														{user.role_name}
													</span>
												</TableCell>
												<TableCell className="text-muted-foreground">
													{user.environment_name}
												</TableCell>
												<TableCell className="text-right">
													<Button
														variant="ghost"
														size="icon"
														onClick={() => handleEdit(user)}
													>
														<Pencil className="h-4 w-4" />
													</Button>
													<Button
														variant="ghost"
														size="icon"
														disabled={generatingLink}
														onClick={() => handleGenerateResetLink(user)}
														title={textGet(
															"backoffice.company_users.password_reset.button_title",
														)}
													>
														<KeyRound className="h-4 w-4" />
													</Button>
												</TableCell>
											</TableRow>
										))}
									</TableBody>
								</Table>
							)}
						</CardContent>
					</Card>
				</div>

				{/* Edit User Dialog */}
				<Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
					<DialogContent className="sm:max-w-md">
						<DialogHeader>
							<DialogTitle>
								{textGet("backoffice.company_users.edit.title")}
							</DialogTitle>
							<DialogDescription>
								{textGet("backoffice.company_users.edit.description")}
							</DialogDescription>
						</DialogHeader>
						{editingUser && (
							<Form<typeof editUserSchema>
								schema={editUserSchema}
								onSubmit={handleSave}
								defaultValues={{
									user_name: editingUser.user_name,
									email: editingUser.email,
									role_id: String(editingUser.role_id),
								}}
							>
								{(field) => (
									<>
										<div className="space-y-4 py-2">
											<FormInput
												field={field}
												name="user_name"
												type="text"
												label={textGet("backoffice.company_users.col.username")}
											/>
											<FormInput
												field={field}
												name="email"
												type="email"
												label={textGet("backoffice.company_users.col.email")}
											/>
											<FormSelect
												field={field}
												name="role_id"
												label={textGet("backoffice.company_users.col.role")}
												placeholder={textGet(
													"backoffice.company_users.select_role",
												)}
												options={roleOptions}
											/>
										</div>
										<DialogFooter>
											<Button
												type="button"
												variant="outline"
												onClick={() => setEditDialogOpen(false)}
											>
												{textGet("backoffice.company_users.cancel")}
											</Button>
											<Button type="submit" disabled={saving}>
												{saving && <Spinner />}
												{textGet("backoffice.company_users.save")}
											</Button>
										</DialogFooter>
									</>
								)}
							</Form>
						)}
					</DialogContent>
				</Dialog>
			</DashboardLayout>

			{/* Password Reset Link Dialog */}
			<Dialog open={resetLinkDialogOpen} onOpenChange={setResetLinkDialogOpen}>
				<DialogContent className="sm:max-w-lg">
					<DialogHeader>
						<DialogTitle className="flex items-center gap-2">
							<KeyRound className="h-4 w-4" />
							{textGet("backoffice.company_users.password_reset.dialog_title")}
						</DialogTitle>
						<DialogDescription>
							{textGet(
								"backoffice.company_users.password_reset.dialog_description",
							)}{" "}
							(<strong>{resetLinkData?.username}</strong>)
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-3 py-2">
						<div className="rounded-md border bg-muted/50 px-3 py-2">
							<p className="break-all text-sm font-mono text-muted-foreground">
								{resetLinkData?.link}
							</p>
						</div>
						<Button
							type="button"
							className="w-full"
							onClick={() =>
								resetLinkData && handleCopyLink(resetLinkData.link)
							}
						>
							<Copy className="mr-2 h-4 w-4" />
							{textGet("backoffice.company_users.password_reset.copy")}
						</Button>
					</div>
					<DialogFooter>
						<Button
							type="button"
							variant="outline"
							className="w-full"
							onClick={() => setResetLinkDialogOpen(false)}
						>
							{textGet("backoffice.company_users.password_reset.close")}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
};

export default CompanyUsers;
