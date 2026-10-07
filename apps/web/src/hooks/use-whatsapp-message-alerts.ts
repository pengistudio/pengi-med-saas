import { useText } from "@pengi/shared";
import * as React from "react";
import { useLocation, useNavigate } from "react-router";
import { toast } from "sonner";
import type { Notification } from "@/api/notification-service";
import { openNotificationLink } from "@/lib/notification-level";
import { playNotificationSound } from "@/lib/notification-sound";
import {
	detectNewWhatsAppMessages,
	isOnWhatsAppInbox,
	notificationTextParams,
	type SeenNotifications,
} from "@/lib/whatsapp-alerts";

/** Message types without text: the toast describes them instead. */
const MEDIA_TYPES = new Set(["image", "audio", "document", "other"]);

/**
 * Returns the handler the notifications poll calls with each page of unread
 * notifications. A WhatsApp message that is new since the previous poll pops a
 * toast ("Abrir" goes to the conversation) and plays a short sound.
 *
 * Exception to "toasts belong to the service layer": this toast isn't the
 * result of a call the user made, it announces something that arrived, so no
 * service call can own it.
 *
 * Nothing pops on the first poll (what is already there isn't news) nor while
 * the user is on the inbox, which shows new messages itself. The sound is
 * skipped when the tab is hidden or the user muted it (inbox header).
 */
export function useWhatsAppMessageAlerts(): (items: Notification[]) => void {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { pathname } = useLocation();
	const seen = React.useRef<SeenNotifications | null>(null);
	// Latest values without changing the handler's identity (the poll's deps).
	const latest = React.useRef({ textGet, navigate, pathname });
	React.useEffect(() => {
		latest.current = { textGet, navigate, pathname };
	}, [textGet, navigate, pathname]);

	return React.useCallback((items: Notification[]) => {
		const result = detectNewWhatsAppMessages(seen.current, items);
		seen.current = result.seen;
		const { textGet, navigate, pathname } = latest.current;
		if (result.fresh.length === 0 || isOnWhatsAppInbox(pathname)) return;
		for (const n of result.fresh) {
			const params = notificationTextParams(n.params);
			const contentType = String(params.content_type ?? "");
			const preview =
				String(params.preview ?? "") ||
				(MEDIA_TYPES.has(contentType)
					? textGet(`whatsapp.inbox.content.${contentType}`)
					: "");
			toast.message(String(params.name ?? ""), {
				id: `whatsapp-${n.ID}`,
				description: preview,
				action: n.action_url
					? {
							label: textGet("whatsapp.alert.open"),
							onClick: () => openNotificationLink(n.action_url, navigate),
						}
					: undefined,
			});
		}
		playNotificationSound();
	}, []);
}
