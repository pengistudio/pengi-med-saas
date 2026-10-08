import React from "react";
import { create } from "zustand";
import {
	type Doctor,
	type DoctorStatus,
	getDoctorStatus,
	getDoctors,
	getSpecialties,
	type Specialty,
} from "@/api/doctors-service";
import { selectEnvironment, useSessionStore } from "@/store/session-store";

// Doctors of the tenant, the current user's onboarding status and the
// specialty catalog. Shared by every doctor selector, the agenda, the notices
// and the admin page, so a screen with several of them fetches each once.
// Keyed by environment (user + tenant): switching tenant or user drops what
// was loaded for the previous one.

type DoctorState = {
	/** Environment (user + tenant) the data belongs to. */
	tenant?: string;
	/** Every doctor, active or not (an inactive one still shows where assigned). */
	doctors: Doctor[];
	doctorsLoaded: boolean;
	status: DoctorStatus | null;
	specialties: Specialty[];
	loadDoctors: (tenant: string, force?: boolean) => Promise<void>;
	loadStatus: (tenant: string, force?: boolean) => Promise<void>;
	loadSpecialties: (tenant: string) => Promise<void>;
	/** Re-fetches doctors and status after a change (admin page, own profile). */
	refresh: () => Promise<void>;
};

type Kind = "doctors" | "status" | "specialties";

// One request per kind and tenant at a time. `seq` numbers the requests of a
// kind: only the latest one writes, so a forced refresh after a mutation is
// never overwritten by an older request still in flight, and a request for a
// previous tenant is never reused nor applied.
const inflight: Partial<
	Record<Kind, { tenant: string; seq: number; promise: Promise<void> }>
> = {};
const latestSeq: Record<Kind, number> = {
	doctors: 0,
	status: 0,
	specialties: 0,
};

function request<T>(
	kind: Kind,
	tenant: string,
	force: boolean,
	fetch: () => Promise<T>,
	apply: (result: T) => void,
	isCurrentTenant: () => boolean,
): Promise<void> {
	const pending = inflight[kind];
	if (pending && pending.tenant === tenant && !force) return pending.promise;
	const seq = ++latestSeq[kind];
	const promise = fetch()
		.then((result) => {
			if (seq === latestSeq[kind] && isCurrentTenant()) apply(result);
		})
		.finally(() => {
			if (inflight[kind]?.seq === seq) inflight[kind] = undefined;
		});
	inflight[kind] = { tenant, seq, promise };
	return promise;
}

const NO_DOCTORS: Doctor[] = [];
const NO_SPECIALTIES: Specialty[] = [];

export const useDoctorStore = create<DoctorState>((set, get) => {
	// Starts over when the tenant changed since the last load.
	const forTenant = (tenant: string) => {
		if (get().tenant !== tenant) {
			set({
				tenant,
				doctors: [],
				doctorsLoaded: false,
				status: null,
				specialties: [],
			});
		}
	};
	const isCurrent = (tenant: string) => () => get().tenant === tenant;

	return {
		tenant: undefined,
		doctors: [],
		doctorsLoaded: false,
		status: null,
		specialties: [],
		loadDoctors: (tenant, force = false) => {
			forTenant(tenant);
			if (get().doctorsLoaded && !force) return Promise.resolve();
			return request(
				"doctors",
				tenant,
				force,
				() => getDoctors({ notifyError: false }),
				(res) =>
					set({
						doctors: res.success && res.data ? res.data : [],
						doctorsLoaded: true,
					}),
				isCurrent(tenant),
			);
		},
		loadStatus: (tenant, force = false) => {
			forTenant(tenant);
			if (get().status && !force) return Promise.resolve();
			return request(
				"status",
				tenant,
				force,
				getDoctorStatus,
				(res) => {
					if (res.success && res.data) set({ status: res.data });
				},
				isCurrent(tenant),
			);
		},
		loadSpecialties: (tenant) => {
			forTenant(tenant);
			if (get().specialties.length > 0) return Promise.resolve();
			return request(
				"specialties",
				tenant,
				false,
				getSpecialties,
				(res) => {
					if (res.success && res.data) set({ specialties: res.data });
				},
				isCurrent(tenant),
			);
		},
		// Always a new request: data from before the mutation must not win.
		refresh: async () => {
			const { tenant } = get();
			if (!tenant) return;
			await Promise.all([
				get().loadDoctors(tenant, true),
				get().loadStatus(tenant, true),
			]);
		},
	};
});

/** The session's environment id (user + tenant), the store key. */
function useEnvironmentKey() {
	const environment = useSessionStore(selectEnvironment);
	return environment ? String(environment.id) : undefined;
}

/** Every doctor of the tenant (active and inactive), loaded on mount. */
export function useDoctors(enabled = true) {
	const tenant = useEnvironmentKey();
	const doctors = useDoctorStore((s) =>
		s.tenant === tenant ? s.doctors : NO_DOCTORS,
	);
	const loaded = useDoctorStore((s) => s.doctorsLoaded && s.tenant === tenant);
	const loadDoctors = useDoctorStore((s) => s.loadDoctors);
	React.useEffect(() => {
		if (enabled && tenant) loadDoctors(tenant);
	}, [enabled, tenant, loadDoctors]);
	const activeDoctors = React.useMemo(
		() => doctors.filter((d) => d.active),
		[doctors],
	);
	return { doctors, activeDoctors, loaded };
}

/** The current user's doctor status (profile, active count, review notice). */
export function useDoctorStatus(enabled = true) {
	const tenant = useEnvironmentKey();
	const status = useDoctorStore((s) => (s.tenant === tenant ? s.status : null));
	const loadStatus = useDoctorStore((s) => s.loadStatus);
	React.useEffect(() => {
		if (enabled && tenant) loadStatus(tenant);
	}, [enabled, tenant, loadStatus]);
	return status;
}

/** The specialty catalog, loaded on mount. */
export function useSpecialties() {
	const tenant = useEnvironmentKey();
	const specialties = useDoctorStore((s) =>
		s.tenant === tenant ? s.specialties : NO_SPECIALTIES,
	);
	const loadSpecialties = useDoctorStore((s) => s.loadSpecialties);
	React.useEffect(() => {
		if (tenant) loadSpecialties(tenant);
	}, [tenant, loadSpecialties]);
	return specialties;
}

/** A doctor's id → doctor lookup over the loaded list. */
export function findDoctor(doctors: Doctor[], id?: number | null) {
	if (!id) return undefined;
	return doctors.find((d) => d.ID === id);
}
