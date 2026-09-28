import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	cn,
	Form,
	FormInput,
	Input,
	Label,
	Spinner,
} from "@pengi/ui";
import { Check, Copy, Link } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import {
	type Company,
	companies,
	companySignupLink,
	getCompanySignupToken,
} from "@/api/company-service";
import { type Plan, plans as planResource } from "@/api/plan-service";
import { subscriptions } from "@/api/subscription-service";
import {
	type PlanTerm,
	PlanTermFields,
} from "@/lib/subscription/plan-term-fields";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const companySchema = z.object({
	trade_name: z.string().min(2),
	legal_name: z.string().min(2),
});

const STEPS = ["empresa", "suscripcion", "acceso"] as const;
type Step = (typeof STEPS)[number];

const StepIndicator = ({
	current,
	textGet,
}: {
	current: Step;
	textGet: (k: string) => string;
}) => {
	const labels: Record<Step, string> = {
		empresa: textGet("backoffice.onboarding.step.company"),
		suscripcion: textGet("backoffice.onboarding.step.subscription"),
		acceso: textGet("backoffice.onboarding.step.access"),
	};
	return (
		<div className="flex items-center justify-center gap-0 mb-8">
			{STEPS.map((step, i) => {
				const idx = STEPS.indexOf(current);
				const done = i < idx;
				const active = i === idx;
				return (
					<React.Fragment key={step}>
						<div className="flex flex-col items-center gap-1">
							<div
								className={cn(
									"w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium border-2 transition-colors",
									done && "bg-primary border-primary text-primary-foreground",
									active && "border-primary text-primary bg-primary/10",
									!done &&
										!active &&
										"border-muted-foreground/30 text-muted-foreground",
								)}
							>
								{done ? <Check className="w-4 h-4" /> : i + 1}
							</div>
							<span
								className={cn(
									"text-xs whitespace-nowrap",
									active ? "text-primary font-medium" : "text-muted-foreground",
								)}
							>
								{labels[step]}
							</span>
						</div>
						{i < STEPS.length - 1 && (
							<div
								className={cn(
									"h-0.5 w-16 mb-4 mx-1 transition-colors",
									i < STEPS.indexOf(current) ? "bg-primary" : "bg-muted",
								)}
							/>
						)}
					</React.Fragment>
				);
			})}
		</div>
	);
};

const CreateCompany = () => {
	const { textGet } = useText();
	const navigate = useNavigate();

	const [step, setStep] = React.useState<Step>("empresa");
	const [loading, setLoading] = React.useState(false);
	const [plans, setPlans] = React.useState<Plan[]>([]);
	const [company, setCompany] = React.useState<Company | null>(null);

	// Step 2 state
	const [term, setTerm] = React.useState<PlanTerm>({
		planCode: "",
		expiresAt: "",
	});

	// Step 3 state
	const [signupLink, setSignupLink] = React.useState("");
	const [copied, setCopied] = React.useState(false);
	const [linkLoading, setLinkLoading] = React.useState(false);

	React.useEffect(() => {
		planResource.list().then((res) => {
			if (res.success && res.data) setPlans(res.data as Plan[]);
		});
	}, []);

	async function handleCompanySubmit(values: z.infer<typeof companySchema>) {
		setLoading(true);
		const res = await companies.create(values);
		setLoading(false);
		if (res.success && res.data) {
			setCompany(res.data as Company);
			setStep("suscripcion");
		}
	}

	async function handleSubscriptionSubmit() {
		if (!company || !term.planCode || !term.expiresAt) return;
		setLoading(true);
		const res = await subscriptions.create({
			company_id: company.ID,
			plan_code: term.planCode,
			status: "active",
			expires_at: term.expiresAt,
		});
		setLoading(false);
		if (res.success) {
			setStep("acceso");
			// Step 3: fetch the signup link as we arrive.
			setLinkLoading(true);
			const linkRes = await getCompanySignupToken(company.ID);
			if (linkRes.success && linkRes.data) {
				setSignupLink(companySignupLink(linkRes.data.token));
			}
			setLinkLoading(false);
		}
	}

	async function handleCopy() {
		await navigator.clipboard.writeText(signupLink);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	}

	return (
		<DashboardLayout>
			<div className="max-w-2xl mx-auto">
				<StepIndicator current={step} textGet={textGet} />

				{step === "empresa" && (
					<Form<typeof companySchema>
						schema={companySchema}
						onSubmit={handleCompanySubmit}
						defaultValues={{ trade_name: "", legal_name: "" }}
					>
						{(field) => (
							<Card>
								<CardHeader>
									<CardTitle>
										{textGet("backoffice.onboarding.company.title")}
									</CardTitle>
									<CardDescription>
										{textGet("backoffice.onboarding.company.description")}
									</CardDescription>
								</CardHeader>
								<CardContent className="space-y-4">
									<FormInput
										field={field}
										name="trade_name"
										type="text"
										label={textGet("backoffice.companies.col.trade_name")}
										placeholder={textGet(
											"backoffice.companies.col.trade_name.placeholder",
										)}
									/>
									<FormInput
										field={field}
										name="legal_name"
										type="text"
										label={textGet("backoffice.companies.col.legal_name")}
										placeholder={textGet(
											"backoffice.companies.col.legal_name.placeholder",
										)}
									/>
								</CardContent>
								<CardFooter className="flex justify-between">
									<Button
										type="button"
										variant="outline"
										onClick={() => navigate("/companies")}
									>
										{textGet("backoffice.companies.cancel")}
									</Button>
									<Button type="submit" disabled={loading}>
										{loading && <Spinner />}
										{textGet("backoffice.onboarding.next")}
									</Button>
								</CardFooter>
							</Card>
						)}
					</Form>
				)}

				{step === "suscripcion" && (
					<Card>
						<CardHeader>
							<CardTitle>
								{textGet("backoffice.onboarding.subscription.title")}
							</CardTitle>
							<CardDescription>
								{textGet("backoffice.onboarding.subscription.description")}
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-4">
							<PlanTermFields plans={plans} value={term} onChange={setTerm} />
						</CardContent>
						<CardFooter className="flex justify-between">
							<Button
								type="button"
								variant="outline"
								onClick={() => setStep("empresa")}
							>
								{textGet("backoffice.onboarding.back")}
							</Button>
							<Button
								onClick={handleSubscriptionSubmit}
								disabled={loading || !term.planCode || !term.expiresAt}
							>
								{loading && <Spinner />}
								{textGet("backoffice.onboarding.next")}
							</Button>
						</CardFooter>
					</Card>
				)}

				{step === "acceso" && (
					<Card>
						<CardHeader>
							<CardTitle className="flex items-center gap-2">
								<Check className="w-5 h-5 text-emerald-500" />
								{textGet("backoffice.onboarding.access.title")}
							</CardTitle>
							<CardDescription>
								{textGet("backoffice.onboarding.access.description")}{" "}
								<strong>{company?.trade_name}</strong>
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-4">
							<div className="space-y-2">
								<Label>
									{textGet("backoffice.companies.signup_link.title")}
								</Label>
								{linkLoading ? (
									<p className="text-sm text-muted-foreground animate-pulse">
										{textGet("backoffice.companies.signup_link.generating")}
									</p>
								) : signupLink ? (
									<div className="flex gap-2">
										<Input
											value={signupLink}
											readOnly
											className="flex-1 text-xs"
										/>
										<Button
											type="button"
											variant="outline"
											size="icon"
											onClick={handleCopy}
										>
											{copied ? (
												<Check className="w-4 h-4 text-emerald-500" />
											) : (
												<Copy className="w-4 h-4" />
											)}
										</Button>
									</div>
								) : (
									<p className="text-sm text-destructive">
										{textGet("backoffice.companies.signup_link.error")}
									</p>
								)}
							</div>
							<p className="text-xs text-muted-foreground flex items-start gap-1.5">
								<Link className="w-3 h-3 mt-0.5 shrink-0" />
								{textGet("backoffice.onboarding.access.hint")}
							</p>
						</CardContent>
						<CardFooter className="flex justify-end">
							<Button onClick={() => navigate("/companies")}>
								{textGet("backoffice.onboarding.finish")}
							</Button>
						</CardFooter>
					</Card>
				)}
			</div>
		</DashboardLayout>
	);
};

export default CreateCompany;
