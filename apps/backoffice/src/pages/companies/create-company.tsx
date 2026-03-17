import { Check, Copy, Link } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import {
	type Company,
	createCompany,
	getCompanySignupToken,
} from "@/api/company-service";
import { getPlans, type Plan } from "@/api/plan-service";
import { createSubscription } from "@/api/subscription-service";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const WEB_APP_URL = import.meta.env.VITE_WEB_APP_URL || "http://localhost:5173";

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
	const [selectedPlanCode, setSelectedPlanCode] = React.useState("");
	const [expiresAt, setExpiresAt] = React.useState("");

	// Step 3 state
	const [signupLink, setSignupLink] = React.useState("");
	const [copied, setCopied] = React.useState(false);
	const [linkLoading, setLinkLoading] = React.useState(false);

	React.useEffect(() => {
		getPlans().then((res) => {
			if (res.success && res.data) setPlans(res.data as Plan[]);
		});
	}, []);

	// Step 3: auto-fetch link when we arrive
	React.useEffect(() => {
		if (step !== "acceso" || !company) return;
		setLinkLoading(true);
		getCompanySignupToken(company.ID).then((res) => {
			if (res.success && res.data) {
				setSignupLink(`${WEB_APP_URL}/signup?token=${res.data.token}`);
			}
			setLinkLoading(false);
		});
	}, [step, company]);

	async function handleCompanySubmit(values: z.infer<typeof companySchema>) {
		setLoading(true);
		const res = await createCompany(values);
		setLoading(false);
		if (res.success && res.data) {
			setCompany(res.data as Company);
			setStep("suscripcion");
		}
	}

	async function handleSubscriptionSubmit() {
		if (!company || !selectedPlanCode || !expiresAt) return;
		setLoading(true);
		const res = await createSubscription({
			company_id: company.ID,
			plan_code: selectedPlanCode,
			status: "active",
			expires_at: new Date(expiresAt).toISOString(),
		});
		setLoading(false);
		if (res.success) {
			setStep("acceso");
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
							<div className="space-y-2">
								<Label>{textGet("backoffice.subscriptions.col.plan")}</Label>
								<Select
									value={selectedPlanCode}
									onValueChange={(val) => setSelectedPlanCode(val ?? "")}
								>
									<SelectTrigger>
										<SelectValue
											placeholder={textGet(
												"backoffice.onboarding.subscription.plan_placeholder",
											)}
										/>
									</SelectTrigger>
									<SelectContent>
										{plans.map((p) => (
											<SelectItem key={p.code} value={p.code}>
												{p.name}{" "}
												<span className="text-muted-foreground text-xs ml-1">
													${p.price}
												</span>
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
							<div className="space-y-2">
								<Label>{textGet("backoffice.subscriptions.col.expires")}</Label>
								<Input
									type="date"
									value={expiresAt}
									onChange={(e) => setExpiresAt(e.target.value)}
									min={new Date().toISOString().split("T")[0]}
								/>
							</div>
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
								disabled={loading || !selectedPlanCode || !expiresAt}
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
