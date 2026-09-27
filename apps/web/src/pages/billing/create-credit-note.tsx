import { Text } from "@pengi/ui";
import { PageHeader } from "@/components/custom/page-header";
import { CreditNoteForm } from "@/sections/forms/billing/credit-note-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const CreateCreditNotePage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4">
				<PageHeader title={<Text uuid="billing.credit_note.create.title" />} />
				<div className="w-full">
					<CreditNoteForm />
				</div>
			</main>
		</DashboardLayout>
	);
};

export default CreateCreditNotePage;
