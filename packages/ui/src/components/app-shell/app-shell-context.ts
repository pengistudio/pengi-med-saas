import { createContext, useContext } from "react";

export interface AppShellState {
	/** Whether nav labels show: always in the phone drawer, per user on the desktop rail. */
	expanded: boolean;
	/** Expands the desktop rail (no-op on phones, where the drawer is already expanded). */
	expand: () => void;
	/** Closes the phone drawer (no-op on desktop). */
	closeDrawer: () => void;
}

// Outside an AppShell (tests, isolated stories) the nav renders expanded.
export const AppShellContext = createContext<AppShellState>({
	expanded: true,
	expand: () => {},
	closeDrawer: () => {},
});

export const useAppShell = () => useContext(AppShellContext);
