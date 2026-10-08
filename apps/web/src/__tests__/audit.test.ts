import { describe, expect, it } from "vitest";
import type { AuditLog } from "@/api/audit-service";
import { auditChanges, formatAuditValue } from "@/lib/audit";

const row = (over: Partial<AuditLog>): AuditLog => ({
	ID: 1,
	tenant_id: 1,
	user_id: 1,
	user_name: "Dra. Ana",
	action: "UPDATE",
	entity_type: "patients",
	entity_id: 7,
	patient_id: 7,
	old_values: null,
	new_values: null,
	created_at: "2026-10-07T10:00:00Z",
	...over,
});

describe("auditChanges", () => {
	it("lists only the fields an update changed, without bookkeeping", () => {
		const changes = auditChanges(
			row({
				old_values: { id: 7, phone: "", email: "a@x.ec", updated_at: "x" },
				new_values: {
					entity_id: 7,
					updated_fields: { phone: "0999", email: "a@x.ec" },
				},
			}),
		);
		expect(changes).toEqual([{ field: "phone", before: "", after: "0999" }]);
	});

	it("shows every value set on a create", () => {
		const changes = auditChanges(
			row({
				action: "CREATE",
				new_values: {
					ID: 3,
					CreatedAt: "x",
					weight: 70,
					blood_pressure: "120/80",
				},
			}),
		);
		expect(changes.map((c) => c.field)).toEqual(["blood_pressure", "weight"]);
	});

	it("has nothing to show for a read", () => {
		expect(auditChanges(row({ action: "READ" }))).toEqual([]);
	});
});

describe("formatAuditValue", () => {
	it("shows empty values as a dash and objects as JSON", () => {
		expect(formatAuditValue(null)).toBe("—");
		expect(formatAuditValue("")).toBe("—");
		expect(formatAuditValue(false)).toBe("false");
		expect(formatAuditValue({ a: 1 })).toBe('{"a":1}');
	});
});
