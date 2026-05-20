import { api } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(api);

export interface PricingOption {
	months: number;
	price: number;
}

export interface Plan {
	ID: number;
	createdAt: string;
	updatedAt: string;
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

export const getPlans = (): Promise<ServiceResponse<Plan[]>> =>
	httpService.get<Plan[]>("/backoffice/plans");

export const getPlanByID = (
	id: number | string,
): Promise<ServiceResponse<Plan>> =>
	httpService.get<Plan>(`/backoffice/plans/${id}`);

export const createPlan = (
	data: CreatePlanRequest,
): Promise<ServiceResponse<Plan>> =>
	httpService.post<Plan>("/backoffice/plans", data, {
		notifySuccess: true,
		notifyError: true,
	});

export const updatePlan = (
	id: number | string,
	data: UpdatePlanRequest,
): Promise<ServiceResponse<Plan>> =>
	httpService.put<Plan>(`/backoffice/plans/${id}`, data, {
		notifySuccess: true,
		notifyError: true,
	});

export const deletePlan = (
	id: number | string,
): Promise<ServiceResponse<null>> =>
	httpService.delete<null>(`/backoffice/plans/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});
