import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { ArrowLeft } from "lucide-react";
import { useNavigate } from "react-router";
import { PageHeader } from "@/components/custom/page-header";
import { WhatsAppSettings } from "@/sections/settings/whatsapp-settings";

/** `/settings/whatsapp`: connection, reminders, templates and the AI bot. */
export default function WhatsAppSettingsPage() {
	const { textGet } = useText();
	const navigate = useNavigate();

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("settings.whatsapp.title")}
				description={textGet("settings.whatsapp.description")}
				actions={
					<Button variant="outline" onClick={() => navigate("/settings")}>
						<ArrowLeft className="mr-2 h-4 w-4" />
						{textGet("settings.whatsapp.back")}
					</Button>
				}
			/>
			<WhatsAppSettings />
		</main>
	);
}
