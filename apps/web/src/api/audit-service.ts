import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from ".";
import type { PaginatedResponse } from "./clinical-service";

const auditService = createHttpService(apiWithTenant);

export type AuditAction = "CREATE" | "UPDATE" | "DELETE" | "READ";

export const AUDIT_ACTIONS: AuditAction[] = [
	"READ",
	"CREATE",
	"UPDATE",
	"DELETE",
];

/** Record types offered in the filter (table names, as the backend stores them). */
export const AUDIT_ENTITY_TYPES = [
	"patients",
	"medical_records",
	"appointments",
	"vital_signs",
	"prescriptions",
	"medical_reports",
	"medical_certificates",
	"exam_orders",
	"patient_attachments",
	"user_signatures",
] as const;

/** One row of the compliance audit trail. */
export interface AuditLog {
	ID: number;
	tenant_id: number;
	user_id: number;
	/** Team name, or user name of someone no longer on the team. */
	user_name: string;
	action: AuditAction;
	/** Table of the record, e.g. "patients". */
	entity_type: string;
	entity_id: number;
	patient_id: number | null;
	patient_name?: string;
	/** Values before the change (UPDATE, DELETE); null otherwise. */
	old_values: Record<string, unknown> | null;
	/** Values after the change (CREATE, UPDATE); null otherwise. */
	new_values: Record<string, unknown> | null;
	created_at: string;
}

export interface AuditUser {
	id: number;
	name: string;
}

export interface AuditLogFilters {
	user_id?: number;
	action?: AuditAction;
	entity_type?: string;
	/** yyyy-mm-dd, inclusive. */
	from?: string;
	/** yyyy-mm-dd, inclusive. */
	to?: string;
	page?: number;
	limit?: number;
}

export const getAuditLogs = async (
	filters: AuditLogFilters = {},
): Promise<ServiceResponse<PaginatedResponse<AuditLog>>> =>
	auditService.get<PaginatedResponse<AuditLog>>("/audit/logs", {
		params: Object.fromEntries(
			Object.entries(filters).filter(([, v]) => v !== undefined && v !== ""),
		),
		notifyError: true,
	});

/** Who appears in the clinic's audit trail (the user filter). */
export const getAuditUsers = async (): Promise<ServiceResponse<AuditUser[]>> =>
	auditService.get<AuditUser[]>("/audit/users", { notifyError: true });
