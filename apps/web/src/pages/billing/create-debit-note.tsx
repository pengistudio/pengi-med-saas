import { Text } from "@pengi/ui";
import { DebitNoteForm } from "@/sections/forms/billing/debit-note-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const CreateDebitNotePage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<div className="flex items-center">
					<h1 className="text-2xl font-semibold tracking-tight">
						<Text uuid="billing.debit_note.create.title" />
					</h1>
				</div>
				<div className="w-full">
					<DebitNoteForm />
				</div>
			</main>
		</DashboardLayout>
	);
};

export default CreateDebitNotePage;
