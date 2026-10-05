import { describe, expect, it, vi } from "vitest";

vi.mock("@/api", () => ({ apiWithTenant: {} }));

import type {
	ExamCatalogItem,
	ExamOrder,
	ExamOrderItem,
	ExamProfile,
} from "@/api/exam-order-service";
import type { PatientAttachment } from "@/api/patient-attachment-service";
import {
	addCatalogItems,
	draftFreeText,
	draftsFromOrder,
	expandProfile,
	groupCatalog,
	itemsPendingReview,
	orderAttachments,
	tabFilters,
	toItemInputs,
} from "@/lib/exam-orders";

const exam = (
	ID: number,
	name: string,
	category: ExamCatalogItem["category"],
	subgroup = "",
	extra: Partial<ExamCatalogItem> = {},
): ExamCatalogItem =>
	({
		ID,
		name,
		category,
		subgroup,
		default_indications: `ind-${ID}`,
		active: true,
		is_default: true,
		sort_order: ID,
		tenant_id: 1,
		...extra,
	}) as ExamCatalogItem;

const catalog = [
	exam(1, "Hemograma", "laboratory", "Hematología"),
	exam(2, "Rx de tórax", "imaging", "Rayos X"),
	exam(3, "Glucosa", "laboratory", "Química sanguínea"),
	exam(4, "Plaquetas", "laboratory", "Hematología"),
	exam(5, "Electrocardiograma", "other"),
];

describe("groupCatalog", () => {
	it("groups by category order, then subgroup, keeping the catalog order", () => {
		const groups = groupCatalog(catalog);
		expect(groups.map((g) => g.category)).toEqual([
			"laboratory",
			"imaging",
			"other",
		]);
		expect(
			groups[0].subgroups.map((s) => [s.subgroup, s.items.map((i) => i.ID)]),
		).toEqual([
			["Hematología", [1, 4]],
			["Química sanguínea", [3]],
		]);
		expect(groups[2].subgroups[0].subgroup).toBe("");
	});

	it("searches name and subgroup ignoring case and accents", () => {
		expect(
			groupCatalog(catalog, "TORAX").flatMap((g) =>
				g.subgroups.flatMap((s) => s.items.map((i) => i.ID)),
			),
		).toEqual([2]);
		expect(
			groupCatalog(catalog, "hematologia").flatMap((g) =>
				g.subgroups.flatMap((s) => s.items.map((i) => i.ID)),
			),
		).toEqual([1, 4]);
		expect(groupCatalog(catalog, "zzz")).toEqual([]);
	});
});

describe("profiles and selection", () => {
	const profile = {
		ID: 1,
		name: "Perfil lipídico",
		active: true,
		items: [
			catalog[0],
			catalog[2],
			exam(9, "Inactivo", "laboratory", "", { active: false }),
		],
	} as ExamProfile;

	it("expands a profile to its active exams with their default indications", () => {
		const drafts = expandProfile([], profile);
		expect(drafts.map((d) => [d.catalog_item_id, d.indications])).toEqual([
			[1, "ind-1"],
			[3, "ind-3"],
		]);
	});

	it("skips exams already selected", () => {
		const start = addCatalogItems([], [catalog[0]]);
		const drafts = expandProfile(start, profile);
		expect(drafts.map((d) => d.catalog_item_id)).toEqual([1, 3]);
		expect(drafts[0]).toBe(start[0]);
	});

	it("maps drafts to item inputs", () => {
		const drafts = [
			...addCatalogItems([], [catalog[1]]),
			draftFreeText("  Prueba X ", "other"),
		];
		drafts[0].indications = "Sin metales";
		expect(toItemInputs(drafts)).toEqual([
			{ catalog_item_id: 2, indications: "Sin metales" },
			{ name: "Prueba X", category: "other", subgroup: "", indications: "" },
		]);
	});
});

const attachment = (ID: number) =>
	({ ID, file_name: `f${ID}.pdf` }) as PatientAttachment;
const item = (
	ID: number,
	attachments: PatientAttachment[] = [],
	reviewed_at: string | null = null,
): ExamOrderItem =>
	({
		ID,
		name: `E${ID}`,
		category: "laboratory",
		subgroup: "",
		indications: "",
		catalog_item_id: ID,
		attachments,
		reviewed_at,
	}) as ExamOrderItem;

describe("orders", () => {
	it("locks the existing exams once the order has results", () => {
		const open = { items: [item(1), item(2)] } as ExamOrder;
		expect(draftsFromOrder(open).every((d) => !d.locked)).toBe(true);
		const withResult = {
			items: [item(1, [attachment(7)]), item(2)],
		} as ExamOrder;
		const drafts = draftsFromOrder(withResult);
		expect(drafts.every((d) => d.locked)).toBe(true);
		expect(toItemInputs(drafts)[0]).toEqual({
			id: 1,
			catalog_item_id: 1,
			indications: "",
		});
	});

	it("lists each result file once and the exams pending review", () => {
		const shared = attachment(7);
		const order = {
			items: [
				item(1, [shared]),
				item(2, [shared, attachment(8)], "2026-10-01T00:00:00Z"),
				item(3),
			],
		} as ExamOrder;
		expect(orderAttachments(order).map((a) => a.ID)).toEqual([7, 8]);
		expect(itemsPendingReview(order).map((i) => i.ID)).toEqual([1]);
	});
});

describe("tabFilters", () => {
	it("maps each tab to the list filters", () => {
		expect(tabFilters("pending_results")).toEqual({
			status: ["issued", "partial_results"],
		});
		expect(tabFilters("to_review")).toEqual({
			pending_review: true,
			ordered_by: "me",
		});
		expect(tabFilters("to_review", true)).toEqual({ pending_review: true });
		expect(tabFilters("all")).toEqual({});
	});
});
