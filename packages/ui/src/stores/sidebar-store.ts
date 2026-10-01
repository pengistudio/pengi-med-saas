import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

interface SidebarState {
	isOpen: boolean;
	toggle: () => void;
	open: () => void;
	close: () => void;
}

// On mobile the sidebar is an overlay that covers the screen, so it always
// starts closed; the persisted preference only applies to the desktop rail.
const isMobile = () =>
	typeof window !== "undefined" &&
	typeof window.matchMedia === "function" &&
	window.matchMedia("(max-width: 767px)").matches;

export const useSidebarStore = create<SidebarState>()(
	persist(
		(set) => ({
			isOpen: !isMobile(),
			toggle: () => set((state) => ({ isOpen: !state.isOpen })),
			open: () => set({ isOpen: true }),
			close: () => set({ isOpen: false }),
		}),
		{
			name: "makari-sidebar-storage",
			storage: createJSONStorage(() => localStorage),
			merge: (persisted, current) =>
				isMobile()
					? current
					: { ...current, ...(persisted as Partial<SidebarState>) },
		},
	),
);
