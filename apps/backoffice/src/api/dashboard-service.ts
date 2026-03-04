import { api } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(api);

export interface DashboardStats {
	total_companies: number;
	total_users: number;
	total_plans: number;
	total_features: number;
	active_subscriptions: number;
	recent_companies: {
		ID: number;
		trade_name: string;
		legal_name: string;
		plan_code: string;
		created_at: string;
		tenant?: { slug: string };
	}[];
}

export const getDashboardStats = (): Promise<ServiceResponse<DashboardStats>> =>
	httpService.get<DashboardStats>("/backoffice/dashboard");
