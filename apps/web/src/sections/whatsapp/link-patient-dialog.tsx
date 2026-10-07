import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
	Text,
} from "@pengi/ui";
import React from "react";
import type { Patient } from "@/api/clinical-service";
import {
	linkWhatsAppConversationPatient,
	type WhatsAppConversationDetail,
} from "@/api/whatsapp-service";
import { PatientSearch } from "./patient-search";

interface LinkPatientDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	conversationId: number;
	phone: string;
	onLinked: (detail: WhatsAppConversationDetail) => void;
}

/** Links an unknown number's conversation to one of the tenant's patients. */
export function LinkPatientDialog({
	open,
	onOpenChange,
	conversationId,
	phone,
	onLinked,
}: LinkPatientDialogProps) {
	const [linkingId, setLinkingId] = React.useState<number | null>(null);

	async function link(patient: Patient) {
		setLinkingId(patient.ID);
		const res = await linkWhatsAppConversationPatient(
			conversationId,
			patient.ID,
		);
		setLinkingId(null);
		if (res.success) {
			onLinked(res.data);
			onOpenChange(false);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>
						<Text uuid="whatsapp.inbox.link.title" />
					</DialogTitle>
					<DialogDescription>
						<Text uuid="whatsapp.inbox.link.description" values={{ phone }} />
					</DialogDescription>
				</DialogHeader>
				{/* Mounted only while open, so closing resets the search. */}
				{open && (
					<PatientSearch
						placeholderKey="whatsapp.inbox.link.search"
						onPick={link}
						pickingId={linkingId}
					/>
				)}
			</DialogContent>
		</Dialog>
	);
}
