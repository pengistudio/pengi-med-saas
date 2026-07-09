import * as React from "react";
import { getNotifications } from "@/api/notification-service";
import { useNotificationStore } from "@/store/notification-store";

const NOTIFICATIONS_POLL_INTERVAL_MS = 30_000;
const RECENT_NOTIFICATIONS_LIMIT = 10;

export function useNotificationsPoll() {
	const setNotifications = useNotificationStore((s) => s.setNotifications);

	const load = React.useCallback(() => {
		getNotifications({ page: 1, limit: RECENT_NOTIFICATIONS_LIMIT }).then(
			(res) => {
				if (res.success) {
					setNotifications(res.data.items, res.data.unread_count);
				}
			},
		);
	}, [setNotifications]);

	React.useEffect(() => {
		load();
	}, [load]);

	React.useEffect(() => {
		const id = setInterval(load, NOTIFICATIONS_POLL_INTERVAL_MS);
		return () => clearInterval(id);
	}, [load]);
}
