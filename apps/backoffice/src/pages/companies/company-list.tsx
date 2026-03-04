import { Pencil, Plus, Trash2 } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	type Company,
	deleteCompany,
	getCompanies,
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
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { useText } from "@/hooks/use-text";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const CompanyList = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [companies, setCompanies] = React.useState<Company[]>([]);
	const [loading, setLoading] = React.useState(true);

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
		</DashboardLayout>
	);
};

export default CompanyList;
