import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	FormInput,
	Spinner,
	Text,
} from "@pengi/ui";
import { Building2 } from "lucide-react";
import { useNavigate } from "react-router";
import z from "zod";
import { createAdditionalCompany } from "@/api/user-service";
import { Form } from "@/components/forms/form";
import { useText } from "@/hooks/use-text";
import { selectSetEnvironment, useSessionStore } from "@/store/session-store";

const formSchema = z.object({
	company_name: z.string().min(2).max(100),
});

const CreateCompanyPage = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const setEnvironment = useSessionStore(selectSetEnvironment);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		const res = await createAdditionalCompany({
			company_name: values.company_name,
		});
		if (res.success && res.data) {
			setEnvironment(res.data);
			navigate("/");
		}
	}

	return (
		<div className="min-h-screen bg-muted/30 flex items-center justify-center p-4">
			<Form<typeof formSchema>
				schema={formSchema}
				onSubmit={onSubmit}
				className="w-full max-w-md"
				defaultValues={{ company_name: "" }}
			>
				{(field) => (
					<Card className="shadow-lg">
						<CardHeader className="text-center space-y-2">
							<div className="flex justify-center mb-2">
								<div className="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center">
									<Building2 className="h-6 w-6 text-primary" />
								</div>
							</div>
							<CardTitle className="text-2xl font-bold">
								<Text uuid="dashboard.new_company.dialog.title" />
							</CardTitle>
							<CardDescription>
								<Text uuid="dashboard.new_company.dialog.description" />
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-4">
							<FormInput
								field={field}
								name="company_name"
								type="text"
								placeholder={textGet(
									"dashboard.new_company.dialog.placeholder",
								)}
								label={textGet("dashboard.new_company.dialog.label")}
							/>
							<Button
								type="submit"
								className="w-full"
								disabled={field.formState.isSubmitting}
							>
								{field.formState.isSubmitting && <Spinner />}
								{textGet("dashboard.new_company.dialog.submit")}
							</Button>
						</CardContent>
					</Card>
				)}
			</Form>
		</div>
	);
};

export default CreateCompanyPage;
