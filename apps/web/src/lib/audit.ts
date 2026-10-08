import type { AuditLog } from "@/api/audit-service";

/** One field of an audited record, before and after the change. */
export interface AuditChange {
	field: string;
	before: unknown;
	after: unknown;
}

/**
 * Bookkeeping columns every record has; they say nothing about the change.
 * Values come as JSON fields (ID, CreatedAt…) or, read back from the table,
 * as column names (id, created_at…).
 */
const NOISE_FIELDS = new Set([
	"ID",
	"CreatedAt",
	"UpdatedAt",
	"DeletedAt",
	"id",
	"created_at",
	"updated_at",
	"deleted_at",
	"tenant_id",
	"entity_id",
	"updated_fields",
]);

/**
 * The fields to show for an audit row: on an update only those that changed,
 * on a create the values set, on a delete the values removed. Reads carry no
 * values. Nested objects (e.g. a record's SOAP) compare as JSON.
 */
export function auditChanges(log: AuditLog): AuditChange[] {
	const before = log.old_values ?? {};
	// Partial updates are stored as {updated_fields: {...}}.
	const rawAfter = log.new_values ?? {};
	const after =
		rawAfter.updated_fields && typeof rawAfter.updated_fields === "object"
			? (rawAfter.updated_fields as Record<string, unknown>)
			: rawAfter;

	const fields = [...new Set([...Object.keys(before), ...Object.keys(after)])]
		.filter((f) => !NOISE_FIELDS.has(f))
		.sort();
	const changes = fields.map((field) => ({
		field,
		before: before[field],
		after: after[field],
	}));
	if (log.action !== "UPDATE") return changes;
	return changes.filter(
		(c) =>
			c.field in after &&
			JSON.stringify(c.before ?? null) !== JSON.stringify(c.after ?? null),
	);
}

/** A value as text: "—" when empty, JSON for objects. */
export function formatAuditValue(value: unknown): string {
	if (value === null || value === undefined || value === "") return "—";
	if (typeof value === "object") return JSON.stringify(value);
	return String(value);
}
