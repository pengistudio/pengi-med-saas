import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { Notification } from "@/api/notification-service";

interface NotificationState {
	notifications: Notification[];
	unreadCount: number;
	setNotifications: (
		notifications: Notification[],
		unreadCount: number,
	) => void;
	markReadLocally: (id: number) => void;
	markAllReadLocally: () => void;
	removeLocally: (id: number, wasUnread: boolean) => void;
	cleanNotifications: () => void;
}

export const useNotificationStore = create<NotificationState>()(
	persist(
		(set) => ({
			notifications: [],
			unreadCount: 0,
			setNotifications: (notifications, unreadCount) =>
				set({ notifications, unreadCount }),
			// The bell only lists unread notifications, so reading one drops
			// it from the list; the full history lives on /notifications.
			// Callers only pass unread notifications, which may be beyond the
			// bell's first page — so always decrement, not only when listed.
			markReadLocally: (id) =>
				set((state) => ({
					notifications: state.notifications.filter((n) => n.ID !== id),
					unreadCount: Math.max(0, state.unreadCount - 1),
				})),
			markAllReadLocally: () => set({ notifications: [], unreadCount: 0 }),
			removeLocally: (id, wasUnread) =>
				set((state) => ({
					notifications: state.notifications.filter((n) => n.ID !== id),
					unreadCount: wasUnread
						? Math.max(0, state.unreadCount - 1)
						: state.unreadCount,
				})),
			cleanNotifications: () => set({ notifications: [], unreadCount: 0 }),
		}),
		{
			// Renaming the key discards any stale cache still sitting in a
			// user's sessionStorage — v2: base fields moved from snake_case
			// (created_at) to BaseModel's CreatedAt; v3: the list holds only
			// unread notifications.
			name: "notification-storage-v3",
			storage: createJSONStorage(() => sessionStorage),
		},
	),
);
