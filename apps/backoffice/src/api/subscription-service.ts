import { api } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(api);

export interface Subscription {
	ID: number;
	createdAt: string;
	updatedAt: string;
	status: string;
	plan_code: string;
	expires_at: string;
	CompanyID: number;
	plan: { ID: number; name: string; code: string; price: number };
}

export interface CreateSubscriptionRequest extends Record<string, unknown> {
	company_id: number;
	plan_code: string;
	status: string;
	expires_at: string;
}

export interface UpdateSubscriptionRequest extends Record<string, unknown> {
	status?: string;
	expires_at?: string;
	plan_code?: string;
}

export const getSubscriptions = (): Promise<ServiceResponse<Subscription[]>> =>
	httpService.get<Subscription[]>("/backoffice/subscriptions");

export const getSubscriptionsByCompany = (
	companyId: number | string,
): Promise<ServiceResponse<Subscription[]>> =>
	httpService.get<Subscription[]>(
		`/backoffice/subscriptions/company/${companyId}`,
	);

export const createSubscription = (
	data: CreateSubscriptionRequest,
): Promise<ServiceResponse<Subscription>> =>
	httpService.post<Subscription>("/backoffice/subscriptions", data, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateSubscription = (
	id: number | string,
	data: UpdateSubscriptionRequest,
): Promise<ServiceResponse<Subscription>> =>
	httpService.put<Subscription>(`/backoffice/subscriptions/${id}`, data, {
		notifySuccess: true,
		notifyError: true,
	});

export const deleteSubscription = (
	id: number | string,
): Promise<ServiceResponse<null>> =>
	httpService.delete<null>(`/backoffice/subscriptions/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});
