import { Text } from "@pengi/ui";
import { PageHeader } from "@/components/custom/page-header";
import { InvoiceForm } from "@/sections/forms/billing/invoice-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const CreateInvoicePage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4">
				<PageHeader title={<Text uuid="billing.invoice.create.title" />} />
				<div className="w-full">
					<InvoiceForm />
				</div>
			</main>
		</DashboardLayout>
	);
};

export default CreateInvoicePage;
