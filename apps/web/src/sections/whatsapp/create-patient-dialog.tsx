import { useText } from "@pengi/shared";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Form,
	FormInput,
	FormRow,
	Text,
} from "@pengi/ui";
import { Loader2 } from "lucide-react";
import React from "react";
import { z } from "zod";
import {
	createWhatsAppConversationPatient,
	type WhatsAppConversationDetail,
} from "@/api/whatsapp-service";

const createPatientSchema = z.object({
	first_name: z
		.string()
		.trim()
		.min(1, "whatsapp.inbox.create_patient.error.first_name"),
	last_name: z.string().trim().optional(),
	document: z.string().trim().optional(),
});

export type CreatePatientValues = z.infer<typeof createPatientSchema>;

interface CreatePatientDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	conversationId: number;
	phone: string;
	/** Pre-fills the first name (e.g. the WhatsApp profile name), when known. */
	defaultName?: string;
	onCreated: (detail: WhatsAppConversationDetail) => void;
}

/**
 * Creates a patient with an unknown number's conversation phone and links the
 * conversation to it. Only the name is asked; the rest is completed later on
 * the patient form.
 */
export function CreatePatientDialog({
	open,
	onOpenChange,
	conversationId,
	phone,
	defaultName = "",
	onCreated,
}: CreatePatientDialogProps) {
	const { textGet } = useText();
	const [loading, setLoading] = React.useState(false);

	async function onSubmit(values: CreatePatientValues) {
		setLoading(true);
		const res = await createWhatsAppConversationPatient(conversationId, {
			first_name: values.first_name,
			last_name: values.last_name || undefined,
			document: values.document || undefined,
		});
		setLoading(false);
		if (res.success) {
			onCreated(res.data);
			onOpenChange(false);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>
						<Text uuid="whatsapp.inbox.create_patient.title" />
					</DialogTitle>
					<DialogDescription>
						<Text
							uuid="whatsapp.inbox.create_patient.description"
							values={{ phone }}
						/>
					</DialogDescription>
				</DialogHeader>
				{/* Mounted only while open, so closing resets the form. */}
				{open && (
					<Form
						schema={createPatientSchema}
						defaultValues={{
							first_name: defaultName,
							last_name: "",
							document: "",
						}}
						onSubmit={onSubmit}
					>
						{(field) => (
							<div className="space-y-4">
								<FormRow>
									<FormInput
										field={field}
										name="first_name"
										label={textGet("whatsapp.inbox.create_patient.first_name")}
										autoFocus
									/>
									<FormInput
										field={field}
										name="last_name"
										label={textGet("whatsapp.inbox.create_patient.last_name")}
										isOptional
									/>
								</FormRow>
								<FormInput
									field={field}
									name="document"
									label={textGet("whatsapp.inbox.create_patient.document")}
									isOptional
								/>
								<DialogFooter>
									<Button type="submit" disabled={loading}>
										{loading && <Loader2 className="animate-spin" />}
										<Text uuid="whatsapp.inbox.create_patient.submit" />
									</Button>
								</DialogFooter>
							</div>
						)}
					</Form>
				)}
			</DialogContent>
		</Dialog>
	);
}
