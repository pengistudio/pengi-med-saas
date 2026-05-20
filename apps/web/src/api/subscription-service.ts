import { apiWithTenant } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const subscriptionService = createHttpService(apiWithTenant);

export interface InitiatePaymentResponse {
	checkout_url: string;
	order_id: string;
	amount: number;
	free: boolean;
	switch_type: "paid" | "immediate" | "deferred";
}

export interface SubscriptionDetail {
	plan_name: string;
	plan_code: string;
	plan_tier: number;
	status: string;
	expires_at: string;
	days_left: number;
	amount: number;
	last_payment_amount: number;
	last_payment_months: number;
	next_plan_code: string;
	plan_change_at: string | null;
}

export interface SubscriptionPaymentRecord {
	ID: number;
	CreatedAt: string;
	order_id: string;
	amount: number;
	status: string;
	checkout_url: string;
}

export interface PricingOption {
	months: number;
	price: number;
}

export interface PlanOption {
	code: string;
	name: string;
	tier: number;
	price: number;
	enabled_features: Record<string, boolean>;
	pricings: PricingOption[];
}

export const getAvailablePlans = async (): Promise<
	ServiceResponse<PlanOption[]>
> =>
	subscriptionService.get<PlanOption[]>("/companies/subscriptions/plans", {
		notifyError: true,
	});

export const initiatePayment = async (
	planCode?: string,
	months?: number,
): Promise<ServiceResponse<InitiatePaymentResponse>> =>
	subscriptionService.post<InitiatePaymentResponse>(
		"/companies/subscriptions/pay",
		{
			...(planCode ? { plan_code: planCode } : {}),
			...(months ? { months } : {}),
		},
		{ notifySuccess: true, notifyError: true },
	);

export const getMySubscription = async (): Promise<
	ServiceResponse<SubscriptionDetail>
> =>
	subscriptionService.get<SubscriptionDetail>("/companies/subscriptions/me", {
		notifyError: true,
	});

export const getSubscriptionPayments = async (): Promise<
	ServiceResponse<SubscriptionPaymentRecord[]>
> =>
	subscriptionService.get<SubscriptionPaymentRecord[]>(
		"/companies/subscriptions/payments",
		{ notifyError: true },
	);

export const confirmPayment = async (): Promise<ServiceResponse<null>> =>
	subscriptionService.post<null>(
		"/companies/subscriptions/confirm-payment",
		{},
		{ notifySuccess: true, notifyError: true },
	);

export const cancelPlanChange = async (): Promise<ServiceResponse<null>> =>
	subscriptionService.delete<null>("/companies/subscriptions/plan-change", {
		notifySuccess: true,
		notifyError: true,
	});
