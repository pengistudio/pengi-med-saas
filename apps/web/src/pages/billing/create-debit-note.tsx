import { Text } from "@pengi/ui";
import { PageHeader } from "@/components/custom/page-header";
import { DebitNoteForm } from "@/sections/forms/billing/debit-note-form";

const CreateDebitNotePage = () => {
	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader title={<Text uuid="billing.debit_note.create.title" />} />
			<div className="w-full">
				<DebitNoteForm />
			</div>
		</main>
	);
};

export default CreateDebitNotePage;
