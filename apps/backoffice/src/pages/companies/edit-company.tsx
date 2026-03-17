import React from "react";
import { useNavigate, useParams } from "react-router";
import z from "zod";
import { getCompanyByID, updateCompany } from "@/api/company-service";
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
});

const EditCompany = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const [loading, setLoading] = React.useState(false);
	const [initialLoading, setInitialLoading] = React.useState(true);
	const [defaultValues, setDefaultValues] = React.useState({
		legal_name: "",
		trade_name: "",
	});

	React.useEffect(() => {
		if (!id) return;
		getCompanyByID(id).then((res) => {
			if (res.success && res.data) {
				const company = res.data;
				setDefaultValues({
					legal_name: company.legal_name,
					trade_name: company.trade_name,
				});
			}
			setInitialLoading(false);
		});
	}, [id]);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!id) return;
		setLoading(true);
		const res = await updateCompany(id, values);
		setLoading(false);
		if (res.success) {
			navigate("/companies");
		}
	}

	if (initialLoading) {
		return (
			<DashboardLayout>
				<div className="flex items-center justify-center h-64">
					<p className="text-muted-foreground animate-pulse">
						{textGet("backoffice.companies.loading")}
					</p>
				</div>
			</DashboardLayout>
		);
	}

	return (
		<DashboardLayout>
			<div className="max-w-2xl mx-auto">
				<Form<typeof formSchema>
					schema={formSchema}
					onSubmit={onSubmit}
					defaultValues={defaultValues}
				>
					{(field) => (
						<Card>
							<CardHeader>
								<CardTitle>
									{textGet("backoffice.companies.edit.title")}
								</CardTitle>
								<CardDescription>
									{textGet("backoffice.companies.edit.description")}
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

export default EditCompany;
