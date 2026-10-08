import React from "react";
import { create } from "zustand";
import {
	type AppointmentType,
	getAppointmentTypes,
} from "@/api/agenda-service";
import { selectEnvironment, useSessionStore } from "@/store/session-store";

// Appointment types of the tenant (active and inactive). Shared by the
// appointment form and the agenda settings page. Keyed by environment (user +
// tenant) like the doctors store: switching tenant drops the previous list.

type AgendaState = {
	tenant?: string;
	types: AppointmentType[];
	typesLoaded: boolean;
	loadTypes: (tenant: string, force?: boolean) => Promise<void>;
	/** Re-fetches the types after a change (settings page). */
	refreshTypes: () => Promise<void>;
};

// One request per tenant at a time; only the latest request writes, so a
// forced refresh after a mutation is never overwritten by an older one.
let inflight: { tenant: string; seq: number; promise: Promise<void> } | null =
	null;
let latestSeq = 0;

const NO_TYPES: AppointmentType[] = [];

export const useAgendaStore = create<AgendaState>((set, get) => ({
	tenant: undefined,
	types: [],
	typesLoaded: false,
	loadTypes: (tenant, force = false) => {
		if (get().tenant !== tenant) {
			set({ tenant, types: [], typesLoaded: false });
		}
		if (get().typesLoaded && !force) return Promise.resolve();
		if (inflight && inflight.tenant === tenant && !force) {
			return inflight.promise;
		}
		const seq = ++latestSeq;
		const promise = getAppointmentTypes()
			.then((res) => {
				if (seq !== latestSeq || get().tenant !== tenant) return;
				// A failed load stays "not loaded", so a later mount retries.
				if (res.success) set({ types: res.data ?? [], typesLoaded: true });
			})
			.finally(() => {
				if (inflight?.seq === seq) inflight = null;
			});
		inflight = { tenant, seq, promise };
		return promise;
	},
	refreshTypes: async () => {
		const { tenant } = get();
		if (tenant) await get().loadTypes(tenant, true);
	},
}));

/** Every appointment type of the tenant, loaded on mount. */
export function useAppointmentTypes(enabled = true) {
	const environment = useSessionStore(selectEnvironment);
	const tenant = environment ? String(environment.id) : undefined;
	const types = useAgendaStore((s) =>
		s.tenant === tenant ? s.types : NO_TYPES,
	);
	const loaded = useAgendaStore((s) => s.typesLoaded && s.tenant === tenant);
	const loadTypes = useAgendaStore((s) => s.loadTypes);
	React.useEffect(() => {
		if (enabled && tenant) loadTypes(tenant);
	}, [enabled, tenant, loadTypes]);
	return { types, loaded };
}
