import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { User } from "@/types/user-type";

type UserState = {
	user?: User;
	clean: () => void;
	setUser: (user: User) => void;
};

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

const persistUser = persist<UserState>(
	(set) => ({
		user: undefined,
		clean: () => set({ user: undefined }),
		setUser: (user: User) => set({ user }),
	}),
	{
		name: "user",
		storage,
	},
);

export const useUserStore = create(persistUser);
