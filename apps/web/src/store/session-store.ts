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
	};
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
			subscriptionExpired: false,
			setSubscriptionExpired: (value: boolean) =>
				set({ subscriptionExpired: value }),
			subscriptionGraceDaysLeft: 0,
			setSubscriptionGraceDaysLeft: (value: number) =>
				set({ subscriptionGraceDaysLeft: value }),
			clean: () =>
				set({
					environment: undefined,
					subscriptionExpired: false,
					subscriptionGraceDaysLeft: 0,
				}),
			setEnvironment: (env: EnvironmentWithCompany) =>
				set({
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
					},
				}),
		}),
		{ name: "session", storage: createJSONStorage(() => localStorage) },
	),
);

// Selectors — pass to useSessionStore(selector) in components
export const selectEnvironment = (s: SessionState) => s.environment;
export const selectSubscriptionExpired = (s: SessionState) =>
	s.subscriptionExpired;
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
