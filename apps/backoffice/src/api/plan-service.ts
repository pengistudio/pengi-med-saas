import { resource } from "@/lib/resource/http-resource";
export interface PricingOption {
	months: number;
	price: number;
}

export interface Plan {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	name: string;
	code: string;
	tier: number;
	price: number;
	Properties: Record<string, unknown>;
	Features: { ID: number; code: string; name: string }[];
	pricings: PricingOption[];
}

export interface CreatePlanRequest extends Record<string, unknown> {
	name: string;
	code: string;
	tier: number;
	price: number;
	properties?: Record<string, unknown>;
	feature_codes?: string[];
	pricings?: PricingOption[];
}

export interface UpdatePlanRequest extends Record<string, unknown> {
	name?: string;
	tier?: number;
	price?: number;
	properties?: Record<string, unknown>;
	feature_codes?: string[];
	pricings?: PricingOption[];
}

export const plans = resource<Plan, CreatePlanRequest, UpdatePlanRequest>(
	"plans",
);
