import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { BackofficeUser } from "@/types/backoffice-user-type";

interface UserState {
	user: BackofficeUser | undefined;
	clean: () => void;
	setUser: (user: BackofficeUser | undefined) => void;
}

const storage = createJSONStorage<UserState>(() => {
	try {
		return sessionStorage;
	} catch {
		return {
			getItem: () => null,
			setItem: () => {},
			removeItem: () => {},
		};
	}
});

export const useUserStore = create<UserState>()(
	persist(
		(set) => ({
			user: undefined,
			clean: () => set({ user: undefined }),
			setUser: (user: BackofficeUser | undefined) => set({ user }),
		}),
		{
			name: "user",
			storage,
		},
	),
);
