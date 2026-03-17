import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

type TokenState = {
	token?: string;
	clean: () => void;
	setToken: (token?: string) => void;
};

const persistToken = persist<TokenState>(
	(set) => ({
		token: undefined,
		clean: () => set({ token: undefined }),
		setToken: (token?: string) => set({ token }),
	}),
	{ name: "token", storage: createJSONStorage(() => localStorage) },
);

export const useTokenStore = create(persistToken);
