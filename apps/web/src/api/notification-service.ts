import { apiWithTenant } from ".";
import {
	type BaseModel,
	createHttpService,
	type ServiceResponse,
} from "./fetch";

const notificationService = createHttpService(apiWithTenant);

export interface Notification extends BaseModel {
	tenant_id: number;
	user_id: number;
	type: string;
	resource_type: string;
	resource_id: number;
	message_key: string;
	params: Record<string, string>;
	action_url: string;
	read_at: string | null;
}

export interface ListNotificationsResponse {
	items: Notification[];
	unread_count: number;
	total: number;
	page: number;
	limit: number;
}

export type NotificationListParams = {
	page?: number;
	limit?: number;
};

export const getNotifications = async (
	params: NotificationListParams = {},
): Promise<ServiceResponse<ListNotificationsResponse>> => {
	const qs = new URLSearchParams();
	if (params.page) qs.set("page", String(params.page));
	if (params.limit) qs.set("limit", String(params.limit));
	const query = qs.toString() ? `?${qs.toString()}` : "";
	// notifyError: false — this is polled silently every 30s, a failed poll
	// shouldn't toast-spam the user.
	return notificationService.get<ListNotificationsResponse>(
		`/notifications${query}`,
		{ notifyError: false },
	);
};

export const markNotificationAsRead = async (
	id: number,
): Promise<ServiceResponse<null>> => {
	return notificationService.patch<null>(
		`/notifications/${id}/read`,
		undefined,
		{ notifyError: false },
	);
};

export const markAllNotificationsAsRead = async (): Promise<
	ServiceResponse<null>
> => {
	return notificationService.post<null>(
		"/notifications/mark-all-read",
		undefined,
		{ notifyError: false },
	);
};
