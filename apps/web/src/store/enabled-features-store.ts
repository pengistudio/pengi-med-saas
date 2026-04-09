import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import {
	DEFAULT_ENABLED_FEATURES,
	type EnabledFeatures,
	getEnabledFeatures,
	updateEnabledFeatures,
} from "@/api/settings-service";

type EnabledFeaturesState = {
	features: EnabledFeatures;
	loaded: boolean;
	fetchFeatures: () => Promise<void>;
	saveFeatures: (features: EnabledFeatures) => Promise<void>;
	reset: () => void;
};

export const useEnabledFeaturesStore = create<EnabledFeaturesState>()(
	persist(
		(set) => ({
			features: DEFAULT_ENABLED_FEATURES,
			loaded: false,

			fetchFeatures: async () => {
				const res = await getEnabledFeatures();
				if (res.success && res.data) {
					set({ features: res.data, loaded: true });
				} else {
					set({ loaded: true });
				}
			},

			saveFeatures: async (newFeatures: EnabledFeatures) => {
				set({ features: newFeatures });
				const res = await updateEnabledFeatures(newFeatures);
				if (res.success && res.data) {
					set({ features: res.data });
				}
			},

			reset: () => set({ features: DEFAULT_ENABLED_FEATURES, loaded: false }),
		}),
		{
			name: "enabled-features",
			storage: createJSONStorage(() => localStorage),
			partialize: (state) => ({ features: state.features }),
		},
	),
);
