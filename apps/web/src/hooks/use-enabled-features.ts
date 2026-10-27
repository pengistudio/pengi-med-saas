import { useMemo } from "react";
import type { EnabledFeatures } from "@/config/nav-config";
import { selectEnvironment, useSessionStore } from "@/store/session-store";

/**
 * The tenant's plan features. Empty when unknown: callers that gate an
 * opt-in feature (e.g. `patient_portal`) check for `=== true`.
 */
export function useEnabledFeatures(): EnabledFeatures {
	const raw = useSessionStore(selectEnvironment)?.enabled_features;
	return useMemo(() => {
		if (!raw) return {};
		try {
			return JSON.parse(raw) as EnabledFeatures;
		} catch {
			return {};
		}
	}, [raw]);
}
