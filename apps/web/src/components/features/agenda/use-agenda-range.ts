import React from "react";
import { type AgendaRange, getAgendaRange } from "@/api/agenda-service";
import { selectEnvironment, useSessionStore } from "@/store/session-store";

/**
 * Working ranges and blocks of the active doctors for the visible range of
 * the agenda (one request per range). null while loading, when `enabled` is
 * off, or when the request failed: the agenda then shades nothing.
 */
export function useAgendaRange(from: string, to: string, enabled: boolean) {
	const [state, setState] = React.useState<{
		key: string;
		data: AgendaRange;
	} | null>(null);
	// The environment (user + tenant) is part of the key: another tenant's
	// ranges are never shown after a switch.
	const environment = useSessionStore(selectEnvironment);
	const key = `${environment?.id ?? ""}|${from}|${to}`;

	React.useEffect(() => {
		if (!enabled) return;
		let current = true;
		getAgendaRange(from, to).then((res) => {
			if (current && res.success && res.data) {
				setState({ key, data: res.data });
			}
		});
		return () => {
			current = false;
		};
	}, [enabled, key, from, to]);

	return enabled && state?.key === key ? state.data : null;
}
