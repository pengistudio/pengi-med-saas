import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import { createCompany } from "@/api/company-service";
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
import { Spinner } from "@/components/ui/spinner";
import { useText } from "@/hooks/use-text";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const formSchema = z.object({
	legal_name: z.string().min(2),
	trade_name: z.string().min(2),
	plan_code: z.string().min(2),
});

const CreateCompany = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [loading, setLoading] = React.useState(false);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoading(true);
		const res = await createCompany(values);
		setLoading(false);
		if (res.success) {
			navigate("/companies");
		}
	}

	return (
		<DashboardLayout>
			<div className="max-w-2xl mx-auto">
				<Form<typeof formSchema>
					schema={formSchema}
					onSubmit={onSubmit}
					defaultValues={{
						legal_name: "",
						trade_name: "",
						plan_code: "",
					}}
				>
					{(field) => (
						<Card>
							<CardHeader>
								<CardTitle>
									{textGet("backoffice.companies.create.title")}
								</CardTitle>
								<CardDescription>
									{textGet("backoffice.companies.create.description")}
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
								<FormInput
									field={field}
									name="plan_code"
									type="text"
									label={textGet("backoffice.companies.col.plan")}
									placeholder="ENTERPRISE, PRO, BASIC..."
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
									{textGet("backoffice.companies.create")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</DashboardLayout>
	);
};

export default CreateCompany;
