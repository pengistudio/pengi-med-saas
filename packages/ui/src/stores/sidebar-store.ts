import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

interface SidebarState {
	/** Whether the desktop rail is expanded. The phone drawer lives in AppShell. */
	isOpen: boolean;
	setOpen: (isOpen: boolean) => void;
}

export const useSidebarStore = create<SidebarState>()(
	persist(
		(set) => ({
			isOpen: true,
			setOpen: (isOpen) => set({ isOpen }),
		}),
		{
			name: "makari-sidebar-storage",
			storage: createJSONStorage(() => localStorage),
		},
	),
);
