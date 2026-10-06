import * as React from "react";
import { getNotifications } from "@/api/notification-service";
import { useWhatsAppMessageAlerts } from "@/hooks/use-whatsapp-message-alerts";
import { useNotificationStore } from "@/store/notification-store";

const NOTIFICATIONS_POLL_INTERVAL_MS = 30_000;
const RECENT_UNREAD_NOTIFICATIONS_LIMIT = 10;

/** Polls the bell's notifications; returns a reload for an early refresh. */
export function useNotificationsPoll(): () => void {
	const setNotifications = useNotificationStore((s) => s.setNotifications);
	const alertWhatsAppMessages = useWhatsAppMessageAlerts();

	const load = React.useCallback(() => {
		getNotifications({
			page: 1,
			limit: RECENT_UNREAD_NOTIFICATIONS_LIMIT,
			unread: true,
		}).then((res) => {
			if (res.success) {
				setNotifications(res.data.items, res.data.unread_count);
				alertWhatsAppMessages(res.data.items ?? []);
			}
		});
	}, [setNotifications, alertWhatsAppMessages]);

	React.useEffect(() => {
		load();
	}, [load]);

	React.useEffect(() => {
		const id = setInterval(load, NOTIFICATIONS_POLL_INTERVAL_MS);
		return () => clearInterval(id);
	}, [load]);

	return load;
}
