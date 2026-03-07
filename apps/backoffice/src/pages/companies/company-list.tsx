import { Check, Copy, Link, Pencil, Plus, Trash2, Users } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	type Company,
	deleteCompany,
	getCompanies,
	getCompanySignupToken,
} from "@/api/company-service";
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
import { Input } from "@/components/ui/input";
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

const WEB_APP_URL = import.meta.env.VITE_WEB_APP_URL || "http://localhost:5173";

const CompanyList = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [companies, setCompanies] = React.useState<Company[]>([]);
	const [loading, setLoading] = React.useState(true);

	// Signup link dialog state
	const [signupDialogOpen, setSignupDialogOpen] = React.useState(false);
	const [signupLink, setSignupLink] = React.useState("");
	const [signupCompanyName, setSignupCompanyName] = React.useState("");
	const [signupLoading, setSignupLoading] = React.useState(false);
	const [copied, setCopied] = React.useState(false);

	const fetchCompanies = React.useCallback(async () => {
		setLoading(true);
		const res = await getCompanies();
		if (res.success && res.data) {
			setCompanies(res.data as Company[]);
		}
		setLoading(false);
	}, []);

	React.useEffect(() => {
		fetchCompanies();
	}, [fetchCompanies]);

	const handleDelete = async (id: number) => {
		const res = await deleteCompany(id);
		if (res.success) {
			fetchCompanies();
		}
	};

	const handleGenerateSignupLink = async (company: Company) => {
		setSignupLoading(true);
		setSignupCompanyName(company.trade_name);
		setSignupDialogOpen(true);
		setCopied(false);

		const res = await getCompanySignupToken(company.ID);
		if (res.success && res.data) {
			setSignupLink(`${WEB_APP_URL}/signup?token=${res.data.token}`);
		} else {
			setSignupLink("");
		}
		setSignupLoading(false);
	};

	const handleCopyLink = async () => {
		if (signupLink) {
			await navigator.clipboard.writeText(signupLink);
			setCopied(true);
			setTimeout(() => setCopied(false), 2000);
		}
	};

	return (
		<DashboardLayout>
			<div className="space-y-6">
				<div className="flex items-center justify-between">
					<h1 className="text-2xl font-bold tracking-tight">
						{textGet("backoffice.companies.title")}
					</h1>
					<Button onClick={() => navigate("/companies/create")}>
						<Plus className="h-4 w-4 mr-2" />
						{textGet("backoffice.companies.create")}
					</Button>
				</div>

				<Card>
					<CardHeader>
						<CardTitle>{textGet("backoffice.companies.list.title")}</CardTitle>
						<CardDescription>
							{textGet("backoffice.companies.list.description")}
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loading ? (
							<p className="text-sm text-muted-foreground py-8 text-center animate-pulse">
								{textGet("backoffice.companies.loading")}
							</p>
						) : companies.length === 0 ? (
							<p className="text-sm text-muted-foreground py-8 text-center">
								{textGet("backoffice.companies.empty")}
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>
											{textGet("backoffice.companies.col.trade_name")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.companies.col.legal_name")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.companies.col.plan")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.companies.col.tenant")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("table.actions")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{companies.map((company) => (
										<TableRow key={company.ID}>
											<TableCell className="font-medium">
												{company.trade_name}
											</TableCell>
											<TableCell>{company.legal_name}</TableCell>
											<TableCell>
												<span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
													{company.plan_code}
												</span>
											</TableCell>
											<TableCell className="text-muted-foreground">
												{company.tenant?.slug}
											</TableCell>
											<TableCell className="text-right">
												<div className="flex items-center justify-end gap-2">
													<Button
														variant="ghost"
														size="icon"
														title={textGet(
															"backoffice.companies.generate_signup_link",
														)}
														onClick={() => handleGenerateSignupLink(company)}
													>
														<Link className="h-4 w-4" />
													</Button>
													<Button
														variant="ghost"
														size="icon"
														title={textGet("backoffice.company_users.title")}
														onClick={() =>
															navigate(`/companies/${company.ID}/users`)
														}
													>
														<Users className="h-4 w-4" />
													</Button>
													<Button
														variant="ghost"
														size="icon"
														onClick={() =>
															navigate(`/companies/edit/${company.ID}`)
														}
													>
														<Pencil className="h-4 w-4" />
													</Button>
													<Button
														variant="ghost"
														size="icon"
														onClick={() => handleDelete(company.ID)}
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

			{/* Signup Link Dialog */}
			<Dialog open={signupDialogOpen} onOpenChange={setSignupDialogOpen}>
				<DialogContent className="sm:max-w-md">
					<DialogHeader>
						<DialogTitle>
							{textGet("backoffice.companies.signup_link.title")}
						</DialogTitle>
						<DialogDescription>
							{textGet("backoffice.companies.signup_link.description")}{" "}
							<strong>{signupCompanyName}</strong>
						</DialogDescription>
					</DialogHeader>
					{signupLoading ? (
						<p className="text-sm text-muted-foreground py-4 text-center animate-pulse">
							{textGet("backoffice.companies.signup_link.generating")}
						</p>
					) : signupLink ? (
						<div className="flex items-center gap-2">
							<Input value={signupLink} readOnly className="flex-1 text-xs" />
							<Button variant="outline" size="icon" onClick={handleCopyLink}>
								{copied ? (
									<Check className="h-4 w-4 text-green-500" />
								) : (
									<Copy className="h-4 w-4" />
								)}
							</Button>
						</div>
					) : (
						<p className="text-sm text-destructive py-4 text-center">
							{textGet("backoffice.companies.signup_link.error")}
						</p>
					)}
					<DialogFooter showCloseButton />
				</DialogContent>
			</Dialog>
		</DashboardLayout>
	);
};

export default CompanyList;
