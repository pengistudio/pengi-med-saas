import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { EnvironmentWithCompany } from "@/types/user-type";

type SessionState = {
	environment?: {
		id: number;
		name: string;
		role: string;
		role_id: number;
		company_id: number;
		legal_name: string;
		trade_name: string;
		plan_code: string;
		tenant_id: number;
		tenant_name: string;
		tenant_slug: string;
		permissions: string[];
		enabled_features?: string;
		/** The clinic requires two-step verification for every user. */
		require_mfa?: boolean;
	};
	/** True once the environment came from the API in this page load (not persisted). */
	permissionsFresh: boolean;
	setPermissionsFresh: (value: boolean) => void;
	/**
	 * A tenant request was refused because the clinic requires two-step
	 * verification and this user has none (E-AUTH-026). Cleared when they
	 * enrol. Not persisted: the next refused request raises it again.
	 */
	mfaRequired: boolean;
	setMfaRequired: (value: boolean) => void;
	/** Updates the clinic's policy in the stored environment after saving it. */
	setRequireMFA: (value: boolean) => void;
	subscriptionExpired: boolean;
	setSubscriptionExpired: (value: boolean) => void;
	subscriptionGraceDaysLeft: number;
	setSubscriptionGraceDaysLeft: (value: number) => void;
	clean: () => void;
	setEnvironment: (env: EnvironmentWithCompany) => void;
};

export const useSessionStore = create<SessionState>()(
	persist(
		(set) => ({
			environment: undefined,
			permissionsFresh: false,
			setPermissionsFresh: (value: boolean) => set({ permissionsFresh: value }),
			mfaRequired: false,
			setMfaRequired: (value: boolean) => set({ mfaRequired: value }),
			setRequireMFA: (value: boolean) =>
				set((s) =>
					s.environment
						? { environment: { ...s.environment, require_mfa: value } }
						: {},
				),
			subscriptionExpired: false,
			setSubscriptionExpired: (value: boolean) =>
				set({ subscriptionExpired: value }),
			subscriptionGraceDaysLeft: 0,
			setSubscriptionGraceDaysLeft: (value: number) =>
				set({ subscriptionGraceDaysLeft: value }),
			clean: () =>
				set({
					environment: undefined,
					permissionsFresh: false,
					mfaRequired: false,
					subscriptionExpired: false,
					subscriptionGraceDaysLeft: 0,
				}),
			setEnvironment: (env: EnvironmentWithCompany) =>
				set({
					permissionsFresh: true,
					environment: {
						id: env.ID,
						name: env.name,
						role: env.role.role,
						role_id: env.role_id,
						company_id: env.company_id,
						legal_name: env.company.legal_name,
						trade_name: env.company.trade_name,
						plan_code: env.company.plan_code,
						tenant_id: env.company.tenant_id,
						tenant_name: env.company.tenant.name,
						tenant_slug: env.company.tenant.slug,
						permissions: env.role.permissions?.map((p) => p.ID) || [],
						enabled_features: env.company.tenant.enabled_features,
						require_mfa: env.company.tenant.require_mfa === true,
					},
				}),
		}),
		{
			name: "session",
			storage: createJSONStorage(() => localStorage),
			partialize: (s) => ({
				environment: s.environment,
				subscriptionExpired: s.subscriptionExpired,
				subscriptionGraceDaysLeft: s.subscriptionGraceDaysLeft,
			}),
		},
	),
);

// Selectors — pass to useSessionStore(selector) in components
export const selectEnvironment = (s: SessionState) => s.environment;
export const selectPermissionsFresh = (s: SessionState) => s.permissionsFresh;
export const selectSubscriptionExpired = (s: SessionState) =>
	s.subscriptionExpired;
export const selectMfaRequired = (s: SessionState) => s.mfaRequired;
export const selectRequireMFA = (s: SessionState) =>
	s.environment?.require_mfa === true;
export const selectTenantSlug = (s: SessionState) => s.environment?.tenant_slug;
export const selectPermissions = (s: SessionState) =>
	s.environment?.permissions ?? [];
export const selectEnabledFeatures = (s: SessionState) =>
	s.environment?.enabled_features;
export const selectSetSubscriptionExpired = (s: SessionState) =>
	s.setSubscriptionExpired;
export const selectSubscriptionGraceDaysLeft = (s: SessionState) =>
	s.subscriptionGraceDaysLeft;
export const selectSetSubscriptionGraceDaysLeft = (s: SessionState) =>
	s.setSubscriptionGraceDaysLeft;
export const selectCleanSession = (s: SessionState) => s.clean;
export const selectSetEnvironment = (s: SessionState) => s.setEnvironment;
