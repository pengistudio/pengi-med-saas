import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	downloadDocument,
	getDocuments,
	type PortalDocument,
} from "@/api/portal-service";
import { saveBlob } from "@/lib/save-blob";
import { DocumentsSection } from "./documents-section";

vi.mock("@/api/portal-service", () => ({
	getDocuments: vi.fn(),
	downloadDocument: vi.fn(),
}));
vi.mock("@/lib/save-blob", () => ({ saveBlob: vi.fn() }));

const ok = <T,>(data: T) => ({
	success: true as const,
	code: 200,
	message: "",
	data,
});
const fail = () => ({ success: false as const, code: 500, message: "" });

const thisYear = new Date().getFullYear();

const doc = (over: Partial<PortalDocument>): PortalDocument => ({
	type: "prescription",
	id: 1,
	date: `${thisYear}-03-04`,
	doctor_name: "ANA PAZ",
	signed_at: `${thisYear}-03-04T10:00:00Z`,
	...over,
});

describe("DocumentsSection", () => {
	beforeEach(() => vi.clearAllMocks());

	it("lists each signed document with its type and doctor", async () => {
		vi.mocked(getDocuments).mockResolvedValue(
			ok([doc({}), doc({ type: "certificate", id: 2, doctor_name: "" })]),
		);
		render(<DocumentsSection />);
		expect(
			await screen.findByText("*portal.documents.type.prescription*"),
		).toBeVisible();
		expect(
			screen.getByText("*portal.documents.type.certificate*"),
		).toBeVisible();
		expect(screen.getByText("Ana Paz")).toBeVisible();
		expect(screen.getByText("*portal.documents.hint*")).toBeVisible();
	});

	it("groups older years under their year", async () => {
		vi.mocked(getDocuments).mockResolvedValue(
			ok([doc({}), doc({ id: 3, date: "2020-01-02" })]),
		);
		render(<DocumentsSection />);
		expect(await screen.findByText("2020")).toBeVisible();
		expect(screen.queryByText(String(thisYear))).toBeNull();
	});

	it("downloads the document by type and id and saves the PDF", async () => {
		const blob = new Blob(["%PDF"]);
		vi.mocked(getDocuments).mockResolvedValue(
			ok([doc({ type: "report", id: 9 })]),
		);
		vi.mocked(downloadDocument).mockResolvedValue(ok(blob));
		const user = userEvent.setup();
		render(<DocumentsSection />);
		await user.click(
			await screen.findByRole("button", { name: /\*portal\.download\*/ }),
		);
		expect(downloadDocument).toHaveBeenCalledWith("report", 9);
		expect(saveBlob).toHaveBeenCalledWith(blob, `report-${thisYear}-03-04.pdf`);
	});

	it("saves nothing when the download fails", async () => {
		vi.mocked(getDocuments).mockResolvedValue(ok([doc({})]));
		vi.mocked(downloadDocument).mockResolvedValue(fail());
		const user = userEvent.setup();
		render(<DocumentsSection />);
		await user.click(
			await screen.findByRole("button", { name: /\*portal\.download\*/ }),
		);
		expect(saveBlob).not.toHaveBeenCalled();
	});

	it("shows the empty and error states", async () => {
		vi.mocked(getDocuments).mockResolvedValueOnce(ok([]));
		const { unmount } = render(<DocumentsSection />);
		expect(await screen.findByText("*portal.documents.empty*")).toBeVisible();
		unmount();
		vi.mocked(getDocuments).mockResolvedValueOnce(fail());
		render(<DocumentsSection />);
		expect(await screen.findByText("*portal.section.error*")).toBeVisible();
	});
});
