import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import {
	DEFAULT_UI_SETTINGS,
	type TenantUISettings,
	getUISettings,
	updateUISettings,
} from "@/api/settings-service";

type TenantSettingsState = {
	settings: TenantUISettings;
	loaded: boolean;
	fetchSettings: () => Promise<void>;
	saveSettings: (settings: TenantUISettings) => Promise<void>;
	reset: () => void;
};

export const useTenantSettingsStore = create<TenantSettingsState>()(
	persist(
		(set) => ({
			settings: DEFAULT_UI_SETTINGS,
			loaded: false,

			fetchSettings: async () => {
				const res = await getUISettings();
				if (res.success && res.data) {
					set({ settings: res.data, loaded: true });
				} else {
					// On API failure keep whatever is in localStorage
					set({ loaded: true });
				}
			},

			saveSettings: async (newSettings: TenantUISettings) => {
				// Optimistic update — persists to localStorage immediately
				set({ settings: newSettings });
				const res = await updateUISettings(newSettings);
				if (res.success && res.data) {
					set({ settings: res.data });
				}
			},

			reset: () => set({ settings: DEFAULT_UI_SETTINGS, loaded: false }),
		}),
		{
			name: "tenant-ui-settings",
			storage: createJSONStorage(() => localStorage),
			// Only persist settings, not loaded — so fetch always runs once per session
			partialize: (state) => ({ settings: state.settings }),
		},
	),
);
