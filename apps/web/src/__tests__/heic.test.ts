import { beforeEach, describe, expect, it, vi } from "vitest";

const heicTo = vi.hoisted(() => vi.fn());
vi.mock("heic-to", () => ({ heicTo }));

import { convertHeicToJpeg, isHeicFile } from "@/lib/heic";

describe("convertHeicToJpeg", () => {
	beforeEach(() => {
		heicTo.mockReset();
		heicTo.mockResolvedValue(new Blob(["jpeg"], { type: "image/jpeg" }));
	});

	it("converts HEIC detected by MIME type", async () => {
		const out = await convertHeicToJpeg(
			new File(["x"], "IMG_1.heic", { type: "image/heic" }),
		);
		expect(heicTo).toHaveBeenCalledOnce();
		expect(out.name).toBe("IMG_1.jpg");
		expect(out.type).toBe("image/jpeg");
	});

	it("converts HEIF detected only by extension (empty type, as iOS Safari)", async () => {
		const out = await convertHeicToJpeg(
			new File(["x"], "foto.HEIF", { type: "" }),
		);
		expect(heicTo).toHaveBeenCalledOnce();
		expect(out.name).toBe("foto.jpg");
		expect(out.type).toBe("image/jpeg");
	});

	it("passes non-HEIC files through untouched", async () => {
		const pdf = new File(["x"], "a.pdf", { type: "application/pdf" });
		expect(await convertHeicToJpeg(pdf)).toBe(pdf);
		expect(heicTo).not.toHaveBeenCalled();
		expect(isHeicFile(pdf)).toBe(false);
	});
});
