import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ upload: vi.fn() }));
vi.mock("@/api/exam-order-service", () => ({
	uploadExamResult: mocks.upload,
}));
vi.mock("@/api/patient-attachment-service", () => ({
	STORAGE_QUOTA_ERROR_CODE: "E-PLAN-003",
	MAX_ATTACHMENT_SIZE: 15 * 1024 * 1024,
	uploadPatientAttachment: vi.fn(),
}));

import { uploadExamResults } from "@/components/features/exam-orders/upload-exam-results";

const pdf = (name: string) =>
	new File(["x"], name, { type: "application/pdf" });

describe("uploadExamResults", () => {
	beforeEach(() => mocks.upload.mockReset());

	it("sends one request per file, each with the selected exams", async () => {
		mocks.upload
			.mockResolvedValueOnce({ success: true, data: { order: { ID: 7 } } })
			.mockResolvedValueOnce({
				success: true,
				data: {
					order: { ID: 7, status: "partial_results" },
					duplicate_of: { id: 3, file_name: "a.pdf", created_at: "2026-10-01" },
				},
			});
		const updates: [number, string | undefined][] = [];
		const result = await uploadExamResults(
			7,
			[pdf("a.pdf"), pdf("b.pdf")],
			{ item_ids: [11, 12], taken_at: "2026-10-01" },
			(id, patch) => updates.push([id, patch.status]),
		);

		expect(mocks.upload).toHaveBeenCalledTimes(2);
		for (const [index, call] of mocks.upload.mock.calls.entries()) {
			expect(call[0]).toBe(7);
			expect(call[1]).toMatchObject({
				item_ids: [11, 12],
				taken_at: "2026-10-01",
			});
			expect(call[1].file.name).toBe(index === 0 ? "a.pdf" : "b.pdf");
			expect(call[2]).toMatchObject({ notify: false });
		}
		expect(result.uploaded).toBe(2);
		expect(result.results.at(-1)?.order.status).toBe("partial_results");
		expect(updates).toContainEqual([0, "done"]);
		expect(updates).toContainEqual([1, "duplicate"]);
	});

	it("stops after the storage quota error", async () => {
		mocks.upload.mockResolvedValue({
			success: false,
			message: "quota",
			data: { error_code: "E-PLAN-003" },
		});
		const result = await uploadExamResults(
			7,
			[pdf("a.pdf"), pdf("b.pdf")],
			{ item_ids: [11] },
			() => {},
		);
		expect(mocks.upload).toHaveBeenCalledTimes(1);
		expect(result).toMatchObject({ failed: 1, quotaReached: true });
	});
});
