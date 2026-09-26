import { useText } from "@pengi/shared";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Input,
} from "@pengi/ui";
import { Check, Copy, Link, Users } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	type Company,
	companies,
	companyRegisterLink,
	companySignupLink,
	generateCompanyRegisterToken,
	getCompanySignupToken,
} from "@/api/company-service";
import { type ResourceColumn, ResourceList } from "@/lib/resource";

const columns: ResourceColumn<Company>[] = [
	{
		header: "backoffice.companies.col.trade_name",
		cell: (c) => c.trade_name,
		className: "font-medium",
	},
	{ header: "backoffice.companies.col.legal_name", cell: (c) => c.legal_name },
	{
		header: "backoffice.companies.col.plan",
		cell: (c) =>
			c.Subscriptions?.[0] ? (
				<span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
					{c.Subscriptions[0].plan_code}
				</span>
			) : (
				<span className="text-muted-foreground text-xs">—</span>
			),
	},
	{
		header: "backoffice.companies.col.tenant",
		cell: (c) => c.tenant?.slug,
		className: "text-muted-foreground",
	},
];

const CompanyList = () => {
	const { textGet } = useText();
	const navigate = useNavigate();

	// Signup link dialog state
	const [signupDialogOpen, setSignupDialogOpen] = React.useState(false);
	const [signupLink, setSignupLink] = React.useState("");
	const [signupCompanyName, setSignupCompanyName] = React.useState("");
	const [linkKind, setLinkKind] = React.useState<"signup" | "register">(
		"signup",
	);
	const [signupLoading, setSignupLoading] = React.useState(false);
	const [copied, setCopied] = React.useState(false);

	const handleGenerateSignupLink = async (company: Company) => {
		setLinkKind("signup");
		setSignupLoading(true);
		setSignupCompanyName(company.trade_name);
		setSignupDialogOpen(true);
		setCopied(false);

		const res = await getCompanySignupToken(company.ID);
		if (res.success && res.data) {
			setSignupLink(companySignupLink(res.data.token));
		} else {
			setSignupLink("");
		}
		setSignupLoading(false);
	};

	const handleGenerateRegisterLink = async () => {
		setLinkKind("register");
		setSignupLoading(true);
		setSignupDialogOpen(true);
		setCopied(false);

		const res = await generateCompanyRegisterToken();
		if (res.success && res.data) {
			setSignupLink(companyRegisterLink(res.data.token));
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
		<>
			<ResourceList
				resource={companies}
				columns={columns}
				itemLabel={(c) => c.trade_name}
				headerActions={
					<Button variant="outline" onClick={handleGenerateRegisterLink}>
						<Link className="h-4 w-4 mr-2" />
						{textGet("backoffice.companies.register_link.button")}
					</Button>
				}
				rowActions={(company) => (
					<>
						<Button
							variant="ghost"
							size="icon"
							title={textGet("backoffice.companies.generate_signup_link")}
							onClick={() => handleGenerateSignupLink(company)}
						>
							<Link className="h-4 w-4" />
						</Button>
						<Button
							variant="ghost"
							size="icon"
							title={textGet("backoffice.company_users.title")}
							onClick={() => navigate(`/companies/${company.ID}/users`)}
						>
							<Users className="h-4 w-4" />
						</Button>
					</>
				)}
			/>

			{/* Signup Link Dialog */}
			<Dialog open={signupDialogOpen} onOpenChange={setSignupDialogOpen}>
				<DialogContent className="sm:max-w-md">
					<DialogHeader>
						<DialogTitle>
							{linkKind === "register"
								? textGet("backoffice.companies.register_link.title")
								: textGet("backoffice.companies.signup_link.title")}
						</DialogTitle>
						<DialogDescription>
							{linkKind === "register" ? (
								textGet("backoffice.companies.register_link.description")
							) : (
								<>
									{textGet("backoffice.companies.signup_link.description")}{" "}
									<strong>{signupCompanyName}</strong>
								</>
							)}
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
		</>
	);
};

export default CompanyList;
