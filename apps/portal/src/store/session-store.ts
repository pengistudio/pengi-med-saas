import { create } from "zustand";

// The portal access token lives in memory only (never localStorage or
// sessionStorage): a reload gets a new one from the httpOnly refresh cookie.
interface SessionStore {
	accessToken: string | undefined;
	setAccessToken: (token: string) => void;
	clear: () => void;
}

export const useSessionStore = create<SessionStore>((set) => ({
	accessToken: undefined,
	setAccessToken: (accessToken) => set({ accessToken }),
	clear: () => set({ accessToken: undefined }),
}));
