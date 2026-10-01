import { Text } from "@pengi/ui";
import { PageHeader } from "@/components/custom/page-header";
import { InvoiceForm } from "@/sections/forms/billing/invoice-form";

const CreateInvoicePage = () => {
	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader title={<Text uuid="billing.invoice.create.title" />} />
			<div className="w-full">
				<InvoiceForm />
			</div>
		</main>
	);
};

export default CreateInvoicePage;
