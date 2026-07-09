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
	cleanNotifications: () => void;
}

export const useNotificationStore = create<NotificationState>()(
	persist(
		(set) => ({
			notifications: [],
			unreadCount: 0,
			setNotifications: (notifications, unreadCount) =>
				set({ notifications, unreadCount }),
			markReadLocally: (id) =>
				set((state) => ({
					notifications: state.notifications.map((n) =>
						n.ID === id ? { ...n, read_at: new Date().toISOString() } : n,
					),
					unreadCount: Math.max(0, state.unreadCount - 1),
				})),
			markAllReadLocally: () =>
				set((state) => ({
					notifications: state.notifications.map((n) => ({
						...n,
						read_at: n.read_at ?? new Date().toISOString(),
					})),
					unreadCount: 0,
				})),
			cleanNotifications: () => set({ notifications: [], unreadCount: 0 }),
		}),
		{
			// v2: the backend used to serialize base fields as snake_case
			// (created_at) before matching the BaseModel convention
			// (CreatedAt); renaming the key discards any old-shaped cache
			// still sitting in a user's sessionStorage.
			name: "notification-storage-v2",
			storage: createJSONStorage(() => sessionStorage),
		},
	),
);
