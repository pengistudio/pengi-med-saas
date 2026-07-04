import { Card, CardContent, CardHeader, CardTitle } from "@pengi/ui";
import { Text } from "@pengi/ui";
import { InvoiceForm } from "@/sections/forms/billing/invoice-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const CreateInvoicePage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<div className="flex items-center">
					<h1 className="text-2xl font-semibold tracking-tight">
						<Text uuid="billing.invoice.create.title" />
					</h1>
				</div>
				<div className="w-full">
					<Card>
						<CardHeader>
							<CardTitle>
								<Text uuid="billing.invoice.create.description" />
							</CardTitle>
						</CardHeader>
						<CardContent>
							<InvoiceForm />
						</CardContent>
					</Card>
				</div>
			</main>
		</DashboardLayout>
	);
};

export default CreateInvoicePage;
