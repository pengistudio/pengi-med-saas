import { Text } from "@pengi/ui";
import { PageHeader } from "@/components/custom/page-header";
import { DebitNoteForm } from "@/sections/forms/billing/debit-note-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const CreateDebitNotePage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4">
				<PageHeader title={<Text uuid="billing.debit_note.create.title" />} />
				<div className="w-full">
					<DebitNoteForm />
				</div>
			</main>
		</DashboardLayout>
	);
};

export default CreateDebitNotePage;
