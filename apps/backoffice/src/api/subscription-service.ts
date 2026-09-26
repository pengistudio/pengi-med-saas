import { resource } from "@/lib/resource/http-resource";
import { api } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(api);

export interface Subscription {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	status: string;
	plan_code: string;
	expires_at: string;
	CompanyID: number;
	company?: { ID: number; trade_name: string; legal_name: string };
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

export const subscriptions = resource<
	Subscription,
	CreateSubscriptionRequest,
	UpdateSubscriptionRequest
>("subscriptions");

export const getSubscriptionsByCompany = (
	companyId: number | string,
): Promise<ServiceResponse<Subscription[]>> =>
	httpService.get<Subscription[]>(
		`/backoffice/subscriptions/company/${companyId}`,
	);
