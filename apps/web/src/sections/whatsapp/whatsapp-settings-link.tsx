import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { Settings } from "lucide-react";
import { useNavigate } from "react-router";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";

/** Shortcut from the inbox to the WhatsApp settings; same permission as the route. */
export function WhatsAppSettingsLink() {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { checkPermission } = usePermission();
	if (!checkPermission([PERMISSIONS.WHATSAPP.PERMISSION_MANAGE_WHATSAPP])) {
		return null;
	}
	const label = textGet("whatsapp.inbox.settings");
	return (
		<Button
			variant="outline"
			aria-label={label}
			title={label}
			onClick={() => navigate("/settings/whatsapp")}
		>
			<Settings />
			<span className="hidden sm:inline">{label}</span>
		</Button>
	);
}
