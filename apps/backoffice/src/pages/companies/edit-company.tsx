import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	Form,
	FormInput,
	Spinner,
} from "@pengi/ui";
import { useNavigate, useParams } from "react-router";
import z from "zod";
import { companies } from "@/api/company-service";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";

const formSchema = z.object({
	legal_name: z.string().min(2),
	trade_name: z.string().min(2),
});

const EditCompany = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const { item, loading, saving, save } = useResourceItem(companies, id);
	const defaultValues = {
		legal_name: item?.legal_name ?? "",
		trade_name: item?.trade_name ?? "",
	};

	async function onSubmit(values: z.infer<typeof formSchema>) {
		await save(values);
	}

	return (
		<ResourceEditPage loading={loading}>
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
									{textGet("backoffice.common.cancel")}
								</Button>
								<Button type="submit" disabled={saving}>
									{saving && <Spinner />}
									{textGet("backoffice.common.save")}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</ResourceEditPage>
	);
};

export default EditCompany;
