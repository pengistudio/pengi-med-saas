import { Text } from "@pengi/ui";
import { CreditNoteForm } from "@/sections/forms/billing/credit-note-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const CreateCreditNotePage = () => {
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<div className="flex items-center">
					<h1 className="text-2xl font-semibold tracking-tight">
						<Text uuid="billing.credit_note.create.title" />
					</h1>
				</div>
				<div className="w-full">
					<CreditNoteForm />
				</div>
			</main>
		</DashboardLayout>
	);
};

export default CreateCreditNotePage;
