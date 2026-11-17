import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { api } from ".";

const httpService = createHttpService(api);

/** One entry of the backoffice audit trail (BackofficeAuditLog in the API). */
export interface BackofficeAuditLog {
	id: number;
	created_at: string;
	backoffice_user_id: number;
	user_name: string;
	method: string;
	/** Registered pattern, e.g. /backoffice/plans/:id. */
	route: string;
	/** URL requested, e.g. /backoffice/plans/3. */
	path: string;
	status: number;
	ip: string;
	user_agent: string;
}

export interface BackofficeAuditLogPage {
	items: BackofficeAuditLog[];
	has_more: boolean;
}

/** Newest first; `beforeId` continues after the last entry already loaded. */
export const getAuditLogs = async (
	beforeId?: number,
): Promise<ServiceResponse<BackofficeAuditLogPage>> =>
	httpService.get<BackofficeAuditLogPage>("/backoffice/audit-logs", {
		params: beforeId ? { before_id: beforeId } : undefined,
		notifyError: true,
	});
