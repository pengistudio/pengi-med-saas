import { useEffect } from "react";
import { useTenantSettingsStore } from "@/store/tenant-settings-store";

/**
 * Provides access to tenant UI settings.
 * Fetches from the API on first use (once per session).
 * Subsequent reads come from localStorage — no extra request.
 */
const useTenantSettings = () => {
	const { settings, loaded, fetchSettings, saveSettings } =
		useTenantSettingsStore();

	useEffect(() => {
		if (!loaded) {
			fetchSettings();
		}
	}, [loaded, fetchSettings]);

	return { settings, saveSettings };
};

export default useTenantSettings;
