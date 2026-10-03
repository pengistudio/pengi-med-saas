import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ upload: vi.fn(), convert: vi.fn() }));
vi.mock("@/api/patient-attachment-service", () => ({
	uploadPatientAttachment: mocks.upload,
	STORAGE_QUOTA_ERROR_CODE: "E-PLAN-003",
	MAX_ATTACHMENT_SIZE: 15 * 1024 * 1024,
}));
vi.mock("@/lib/heic", async (orig) => ({
	...(await orig<typeof import("@/lib/heic")>()),
	convertHeicToJpeg: mocks.convert,
}));

import { uploadBatch } from "@/components/features/patient-attachments/upload-batch";

const pdf = (name: string) =>
	new File(["x"], name, { type: "application/pdf" });
const meta = { category: "other" as const };

describe("uploadBatch", () => {
	beforeEach(() => {
		mocks.upload.mockReset();
		mocks.convert.mockReset();
	});

	it("keeps going after a failure and reports each file", async () => {
		mocks.upload
			.mockResolvedValueOnce({
				success: false,
				message: "boom",
				data: { error_code: "E-X" },
			})
			.mockResolvedValueOnce({ success: true, data: { ID: 9 } });
		const updates: [number, string | undefined][] = [];
		const r = await uploadBatch(
			7,
			[pdf("a.pdf"), pdf("b.pdf")],
			meta,
			(id, p) => updates.push([id, p.status]),
		);
		expect(r).toEqual({
			uploaded: 1,
			failed: 1,
			quotaReached: false,
			attachments: [{ ID: 9 }],
		});
		expect(updates).toContainEqual([0, "error"]);
		expect(updates).toContainEqual([1, "done"]);
	});

	it("rejects invalid files without uploading them", async () => {
		mocks.upload.mockResolvedValue({ success: true, data: {} });
		const r = await uploadBatch(
			7,
			[
				new File(["x"], "a.exe", { type: "application/x-msdownload" }),
				pdf("b.pdf"),
			],
			meta,
			() => {},
		);
		expect(mocks.upload).toHaveBeenCalledTimes(1);
		expect(r.failed).toBe(1);
	});

	it("stops the remaining files on a quota error", async () => {
		mocks.upload.mockResolvedValue({
			success: false,
			message: "quota",
			data: { error_code: "E-PLAN-003" },
		});
		const updates: [number, string | undefined][] = [];
		const r = await uploadBatch(
			7,
			[pdf("a.pdf"), pdf("b.pdf"), pdf("c.pdf")],
			meta,
			(id, p) => updates.push([id, p.status]),
		);
		expect(mocks.upload).toHaveBeenCalledTimes(1);
		expect(r.quotaReached).toBe(true);
		expect(updates).toContainEqual([2, "skipped"]);
	});

	it("converts HEIC before uploading the JPEG", async () => {
		const jpg = new File(["j"], "i.jpg", { type: "image/jpeg" });
		mocks.convert.mockResolvedValue(jpg);
		mocks.upload.mockResolvedValue({ success: true, data: {} });
		await uploadBatch(
			7,
			[new File(["h"], "i.heic", { type: "" })],
			meta,
			() => {},
		);
		expect(mocks.upload.mock.calls[0][1].file).toBe(jpg);
	});
});
