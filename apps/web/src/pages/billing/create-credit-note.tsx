import { Text } from "@pengi/ui";
import { PageHeader } from "@/components/custom/page-header";
import { CreditNoteForm } from "@/sections/forms/billing/credit-note-form";

const CreateCreditNotePage = () => {
	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader title={<Text uuid="billing.credit_note.create.title" />} />
			<div className="w-full">
				<CreditNoteForm />
			</div>
		</main>
	);
};

export default CreateCreditNotePage;
