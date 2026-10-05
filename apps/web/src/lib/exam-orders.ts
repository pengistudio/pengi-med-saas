import {
	EXAM_CATEGORIES,
	type ExamCatalogItem,
	type ExamCategory,
	type ExamOrder,
	type ExamOrderFilters,
	type ExamOrderItem,
	type ExamOrderItemInput,
	type ExamOrderStatus,
	type ExamPriority,
	type ExamProfile,
} from "@/api/exam-order-service";
import type { PatientAttachment } from "@/api/patient-attachment-service";

// Pure helpers of the exam orders screens (no React), kept apart so the
// picker grouping, the profile expansion and the list tabs are unit-tested.

/** i18n keys of the order statuses. */
export const EXAM_ORDER_STATUS_KEYS: Record<ExamOrderStatus, string> = {
	issued: "clinical.exam_orders.status.issued",
	partial_results: "clinical.exam_orders.status.partial_results",
	complete_results: "clinical.exam_orders.status.complete_results",
	voided: "clinical.exam_orders.status.voided",
};

/** i18n keys of the exam categories. */
export const EXAM_CATEGORY_KEYS: Record<ExamCategory, string> = {
	laboratory: "clinical.exam_orders.category.laboratory",
	imaging: "clinical.exam_orders.category.imaging",
	other: "clinical.exam_orders.category.other",
};

/** i18n keys of the priorities. */
export const EXAM_PRIORITY_KEYS: Record<ExamPriority, string> = {
	routine: "clinical.exam_orders.priority.routine",
	urgent: "clinical.exam_orders.priority.urgent",
};

// ─── Catalog grouping ───────────────────────────────────────────────────────

export interface CatalogSubgroup {
	/** "" for exams without a subgroup. */
	subgroup: string;
	items: ExamCatalogItem[];
}

export interface CatalogGroup {
	category: ExamCategory;
	subgroups: CatalogSubgroup[];
}

/** Lowercase without accents, so "hemoglobina" finds "Hemoglobína". */
export const normalizeSearch = (value: string) =>
	value.normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase().trim();

/**
 * Groups the catalog by category (laboratory, imaging, other) and subgroup,
 * keeping the server's order inside each group. With a query, keeps the exams
 * whose name or subgroup contains it. Empty groups are dropped.
 */
export function groupCatalog(
	items: ExamCatalogItem[],
	query = "",
): CatalogGroup[] {
	const q = normalizeSearch(query);
	const matches = (item: ExamCatalogItem) =>
		!q ||
		normalizeSearch(item.name).includes(q) ||
		normalizeSearch(item.subgroup ?? "").includes(q);
	const groups: CatalogGroup[] = [];
	for (const category of EXAM_CATEGORIES) {
		const subgroups: CatalogSubgroup[] = [];
		const bySubgroup = new Map<string, CatalogSubgroup>();
		for (const item of items) {
			const itemCategory: ExamCategory = EXAM_CATEGORIES.includes(item.category)
				? item.category
				: "other";
			if (itemCategory !== category || !matches(item)) continue;
			const name = item.subgroup ?? "";
			let group = bySubgroup.get(name);
			if (!group) {
				group = { subgroup: name, items: [] };
				bySubgroup.set(name, group);
				subgroups.push(group);
			}
			group.items.push(item);
		}
		if (subgroups.length > 0) groups.push({ category, subgroups });
	}
	return groups;
}

/** The distinct subgroups of a category, in catalog order. */
export function catalogSubgroups(
	items: ExamCatalogItem[],
	category?: ExamCategory,
): string[] {
	// A Set keeps insertion order: the catalog's.
	const seen = new Set<string>();
	for (const item of items) {
		if (category && item.category !== category) continue;
		if (item.subgroup) seen.add(item.subgroup);
	}
	return [...seen];
}

// ─── Draft items (the order form) ───────────────────────────────────────────

/** One exam while the order is being written. */
export interface DraftExamItem {
	/** Stable React key. */
	key: string;
	/** Existing order item (edit). */
	id?: number;
	catalog_item_id?: number;
	name: string;
	category: ExamCategory;
	subgroup: string;
	indications: string;
	/** The order has results: existing exams can't change or be removed. */
	locked: boolean;
}

let draftCounter = 0;
const nextKey = (prefix: string) => `${prefix}-${++draftCounter}`;

export function draftFromCatalog(item: ExamCatalogItem): DraftExamItem {
	return {
		key: nextKey(`catalog-${item.ID}`),
		catalog_item_id: item.ID,
		name: item.name,
		category: item.category,
		subgroup: item.subgroup ?? "",
		indications: item.default_indications ?? "",
		locked: false,
	};
}

/** An exam written by hand (not in the catalog). */
export function draftFreeText(
	name: string,
	category: ExamCategory,
): DraftExamItem {
	return {
		key: nextKey("free"),
		name: name.trim(),
		category,
		subgroup: "",
		indications: "",
		locked: false,
	};
}

/**
 * Adds catalog exams to the selection, skipping inactive ones and those
 * already selected (an exam is ordered once per order).
 */
export function addCatalogItems(
	drafts: DraftExamItem[],
	items: ExamCatalogItem[],
): DraftExamItem[] {
	const selected = new Set(
		drafts.map((d) => d.catalog_item_id).filter((id) => id !== undefined),
	);
	const added: DraftExamItem[] = [];
	for (const item of items) {
		if (!item.active || selected.has(item.ID)) continue;
		selected.add(item.ID);
		added.push(draftFromCatalog(item));
	}
	return added.length > 0 ? [...drafts, ...added] : drafts;
}

/** "Add profile": expands to the profile's exams not selected yet. */
export const expandProfile = (drafts: DraftExamItem[], profile: ExamProfile) =>
	addCatalogItems(drafts, profile.items ?? []);

/** Whether any exam of the order has a result. */
export const hasAnyResult = (order: Pick<ExamOrder, "items">) =>
	(order.items ?? []).some((item) => (item.attachments ?? []).length > 0);

/** The order's exams as drafts; with results, they are locked. */
export function draftsFromOrder(order: ExamOrder): DraftExamItem[] {
	const locked = hasAnyResult(order);
	return (order.items ?? []).map((item) => ({
		key: `item-${item.ID}`,
		id: item.ID,
		catalog_item_id: item.catalog_item_id ?? undefined,
		name: item.name,
		category: item.category,
		subgroup: item.subgroup ?? "",
		indications: item.indications ?? "",
		locked,
	}));
}

/** The drafts as the API's item inputs. */
export function toItemInputs(drafts: DraftExamItem[]): ExamOrderItemInput[] {
	return drafts.map((d) => {
		const input: ExamOrderItemInput = { indications: d.indications };
		if (d.id !== undefined) input.id = d.id;
		if (d.catalog_item_id !== undefined) {
			input.catalog_item_id = d.catalog_item_id;
		} else {
			input.name = d.name;
			input.category = d.category;
			input.subgroup = d.subgroup;
		}
		return input;
	});
}

// ─── Status, results and review ─────────────────────────────────────────────

export const isOrderVoided = (order: Pick<ExamOrder, "status" | "voided_at">) =>
	order.status === "voided" || Boolean(order.voided_at);

export const itemHasResult = (item: ExamOrderItem) =>
	(item.attachments ?? []).length > 0;

/** Exams with a result nobody reviewed yet. */
export const itemsPendingReview = (order: Pick<ExamOrder, "items">) =>
	(order.items ?? []).filter(
		(item) => itemHasResult(item) && !item.reviewed_at,
	);

/** The exams a result file covers. */
export const itemsOfAttachment = (
	order: Pick<ExamOrder, "items">,
	attachmentId: number,
) =>
	(order.items ?? []).filter((item) =>
		(item.attachments ?? []).some((a) => a.ID === attachmentId),
	);

/** Every result file of the order once, in exam order (viewer navigation). */
export function orderAttachments(
	order: Pick<ExamOrder, "items">,
): PatientAttachment[] {
	const seen = new Set<number>();
	const out: PatientAttachment[] = [];
	for (const item of order.items ?? []) {
		for (const attachment of item.attachments ?? []) {
			if (seen.has(attachment.ID)) continue;
			seen.add(attachment.ID);
			out.push(attachment);
		}
	}
	return out;
}

// ─── List tabs ──────────────────────────────────────────────────────────────

export const EXAM_ORDER_TABS = ["pending_results", "to_review", "all"] as const;
export type ExamOrderTab = (typeof EXAM_ORDER_TABS)[number];

/**
 * The list filters of a tab. "Por revisar" shows the current doctor's orders
 * unless `allDoctors` is on.
 */
export function tabFilters(
	tab: ExamOrderTab,
	allDoctors = false,
): ExamOrderFilters {
	switch (tab) {
		case "pending_results":
			return { status: ["issued", "partial_results"] };
		case "to_review":
			return allDoctors
				? { pending_review: true }
				: { pending_review: true, ordered_by: "me" };
		case "all":
			return {};
	}
}

export const isExamOrderTab = (value: unknown): value is ExamOrderTab =>
	EXAM_ORDER_TABS.includes(value as ExamOrderTab);

/** The patient's name as the order screens show it. */
export const patientDisplayName = (
	patient?: {
		full_name?: string;
		first_name?: string;
		last_name?: string;
	} | null,
) =>
	patient
		? patient.full_name ||
			`${patient.first_name ?? ""} ${patient.last_name ?? ""}`.trim()
		: "";
