import { beforeEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => {
	const ok = (data: unknown = null) =>
		Promise.resolve({ status: 200, data: { code: 200, message: "ok", data } });
	return {
		get: vi.fn(() => ok()),
		post: vi.fn(() => ok()),
		put: vi.fn(() => ok()),
		patch: vi.fn(() => ok()),
		delete: vi.fn(() => ok()),
	};
});
vi.mock("@/api", () => ({
	apiWithTenant: client,
	api: client,
	noAuthApi: client,
}));
vi.mock("sonner", () => ({
	toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

import {
	closeExamOrder,
	createExamOrder,
	deleteExamResult,
	emailExamOrder,
	examOrderQuery,
	getExamCatalog,
	getExamOrders,
	restoreExamCatalogDefaults,
	reviewExamResults,
	updateExamProfile,
	uploadExamResult,
	voidExamOrder,
} from "@/api/exam-order-service";

describe("exam-order-service", () => {
	beforeEach(() => {
		for (const fn of Object.values(client)) fn.mockClear();
	});

	it("builds the list query without empty filters", () => {
		expect(examOrderQuery({})).toEqual({});
		expect(
			examOrderQuery({
				status: ["issued", "partial_results"],
				pending_review: true,
				ordered_by: "me",
				patient_id: 4,
				record_id: 9,
				page: 2,
				limit: 20,
			}),
		).toEqual({
			status: "issued,partial_results",
			pending_review: "true",
			ordered_by: "me",
			patient_id: 4,
			record_id: 9,
			page: 2,
			limit: 20,
		});
		expect(examOrderQuery({ status: [], pending_review: false })).toEqual({});
	});

	it("lists orders with the query params", async () => {
		await getExamOrders({ pending_review: true, ordered_by: "me" });
		expect(client.get).toHaveBeenCalledWith(
			"/clinical/exam-orders",
			expect.objectContaining({
				params: { pending_review: "true", ordered_by: "me" },
			}),
		);
	});

	it("filters the catalog", async () => {
		await getExamCatalog({ active: true, category: "imaging", q: " rx " });
		expect(client.get).toHaveBeenCalledWith(
			"/clinical/exam-catalog",
			expect.objectContaining({
				params: { active: "true", category: "imaging", q: "rx" },
			}),
		);
	});

	it("creates an order with its items", async () => {
		const payload = {
			patient_id: 3,
			medical_record_id: 8,
			diagnoses: [{ code: "E11", title: "Diabetes" }],
			priority: "urgent" as const,
			notes: "",
			destination_lab: "Lab",
			items: [
				{ catalog_item_id: 5, indications: "Ayuno" },
				{ name: "Libre", category: "other" as const, subgroup: "" },
			],
		};
		await createExamOrder(payload);
		expect(client.post).toHaveBeenCalledWith(
			"/clinical/exam-orders",
			payload,
			expect.anything(),
		);
	});

	it("uses the order action routes", async () => {
		await voidExamOrder(7, "error");
		expect(client.post).toHaveBeenLastCalledWith(
			"/clinical/exam-orders/7/void",
			{ reason: "error" },
			expect.anything(),
		);
		await closeExamOrder(7);
		expect(client.post).toHaveBeenLastCalledWith(
			"/clinical/exam-orders/7/close",
			undefined,
			expect.anything(),
		);
		await emailExamOrder(7, "a@b.ec");
		expect(client.post).toHaveBeenLastCalledWith(
			"/clinical/exam-orders/7/email",
			{ email: "a@b.ec" },
			expect.anything(),
		);
		await reviewExamResults(7, [1, 2]);
		expect(client.post).toHaveBeenLastCalledWith(
			"/clinical/exam-orders/7/review",
			{ item_ids: [1, 2] },
			expect.anything(),
		);
		await restoreExamCatalogDefaults();
		expect(client.post).toHaveBeenLastCalledWith(
			"/clinical/exam-catalog/restore-defaults",
			undefined,
			expect.anything(),
		);
	});

	it("deletes a result with its reason in the body", async () => {
		await deleteExamResult(7, 30, "duplicado");
		expect(client.delete).toHaveBeenCalledWith(
			"/clinical/exam-orders/7/results/30",
			expect.objectContaining({ data: { reason: "duplicado" } }),
		);
	});

	it("updates a profile's exams", async () => {
		await updateExamProfile(2, { item_ids: [4, 5] });
		expect(client.put).toHaveBeenCalledWith(
			"/clinical/exam-profiles/2",
			{ item_ids: [4, 5] },
			expect.anything(),
		);
	});

	it("uploads one file per request with each item id", async () => {
		const file = new File(["x"], "hemograma.pdf", { type: "application/pdf" });
		await uploadExamResult(7, {
			file,
			item_ids: [11, 12],
			taken_at: "2026-10-01",
		});
		const [url, form] = client.post.mock.calls[0] as unknown as [
			string,
			FormData,
		];
		expect(url).toBe("/clinical/exam-orders/7/results");
		expect(form.get("file")).toBe(file);
		expect(form.getAll("item_ids")).toEqual(["11", "12"]);
		expect(form.get("taken_at")).toBe("2026-10-01");
		expect(form.has("category")).toBe(false);
		expect(form.has("description")).toBe(false);
	});
});
