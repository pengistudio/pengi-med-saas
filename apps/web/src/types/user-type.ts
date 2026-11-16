import type { Base } from "./api-type";
import type { Permission } from "./permission-type";

export type User = {
	id: number;
	user_name: string;
	email: string;
	Environments: Environment[];
} & Base;

export type Environment = {
	name: string;
	role_id: number;
	role: Role;
	company_id: number;
	company: Company;
} & Base;

export type Tenant = {
	name: string;
	slug: string;
	plan_code: string;
	enabled_features?: string; // JSON-encoded EnabledFeatures
	require_mfa?: boolean; // the clinic requires two-step verification
} & Base;

export type Company = {
	legal_name: string;
	trade_name: string;
	plan_code: string;
	tenant_id: number;
	tenant: Tenant;
} & Base;

export type EnvironmentWithCompany = Environment & {
	company: Company;
};

export type Role = {
	role: string;
	permissions: Permission[];
} & Base;
// Inline types based on usage if not defined elsewhere
export type LoginRequest = {
	user_name: string;
	password: string;
};

export type LoginResponse = {
	token: string;
	user_id: number;
	exchange_token: string;
};

/** A correct password of a user with 2FA: no session yet, a code is needed. */
export type TwoFactorChallenge = {
	two_factor_required: true;
	mfa_token: string;
};

/** What /auth/login (and /auth/signup/company/join) answers on success. */
export type LoginResult = LoginResponse | TwoFactorChallenge;

export const isTwoFactorChallenge = (
	data: LoginResult,
): data is TwoFactorChallenge =>
	(data as TwoFactorChallenge).two_factor_required === true;
