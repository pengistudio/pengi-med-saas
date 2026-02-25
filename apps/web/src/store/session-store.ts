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
	};
	clean: () => void;
	setEnvironment: (env: EnvironmentWithCompany) => void;
};

const persistSession = persist<SessionState>(
	(set) => ({
		environment: undefined,
		clean: () => set({ environment: undefined }),
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
				},
			}),
	}),
	{ name: "session", storage: createJSONStorage(() => sessionStorage) },
);

export const useSessionStore = create(persistSession);
