import { Button, Text } from "@pengi/ui";
import { MessagesSquare } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import type { Patient } from "@/api/clinical-service";
import { getWhatsAppConversations } from "@/api/whatsapp-service";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import {
	recipientFromPatient,
	TemplateMessageDialog,
} from "@/sections/whatsapp/template-message-dialog";

/**
 * "Escribir por WhatsApp" on the patient's page. With a conversation in the
 * inbox (linked to the patient, or with their phone) it opens it; without one
 * it opens "Nuevo mensaje" with the patient already chosen.
 */
export function OpenChatButton({ patient }: { patient: Patient }) {
	const navigate = useNavigate();
	const { checkPermission } = usePermission();
	const canUseInbox = checkPermission([
		PERMISSIONS.WHATSAPP.PERMISSION_USE_WHATSAPP_INBOX,
	]);
	const patientId = patient.ID;
	const phone = patient.phone;
	const [found, setFound] = React.useState<{
		patientId: number;
		conversationId: number | null;
	} | null>(null);
	const [dialogOpen, setDialogOpen] = React.useState(false);

	React.useEffect(() => {
		if (!canUseInbox || !phone) return;
		let cancelled = false;
		// Silent: a plan without the inbox just hides the button.
		getWhatsAppConversations({ search: phone, limit: 5 }).then((res) => {
			if (cancelled) return;
			const items = res.success ? (res.data.items ?? []) : [];
			const match =
				items.find((c) => c.patient_id === patientId) ?? items[0] ?? null;
			setFound({ patientId, conversationId: match?.id ?? null });
		});
		return () => {
			cancelled = true;
		};
	}, [canUseInbox, phone, patientId]);

	// Ignore a lookup made for another patient; wait for this one's.
	const lookup = found?.patientId === patientId ? found : null;
	if (!canUseInbox || !phone || !lookup) return null;
	const conversationId = lookup.conversationId;

	return (
		<>
			<Button
				variant="outline"
				onClick={() =>
					conversationId
						? navigate(`/whatsapp?c=${conversationId}`)
						: setDialogOpen(true)
				}
			>
				<MessagesSquare />
				<Text uuid="whatsapp.patient.write" />
			</Button>
			{!conversationId && (
				<TemplateMessageDialog
					open={dialogOpen}
					onOpenChange={setDialogOpen}
					patient={recipientFromPatient(patient)}
					onSent={({ conversation }) =>
						navigate(`/whatsapp?c=${conversation.id}`)
					}
				/>
			)}
		</>
	);
}
