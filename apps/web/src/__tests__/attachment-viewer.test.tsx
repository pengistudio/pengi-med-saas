import { UiTextProvider } from "@pengi/ui";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ viewPatientAttachment: vi.fn() }));
vi.mock("@/api/patient-attachment-service", () => mocks);

vi.mock("@/components/features/patient-attachments/pdf-pages", () => ({
	default: ({ blob }: { blob: Blob }) => (
		<div data-testid="pdf-pages">{blob.type}</div>
	),
}));

import { AttachmentViewer } from "@/components/features/patient-attachments/attachment-viewer";

const base = {
	ID: 3,
	CreatedAt: "",
	UpdatedAt: "",
	DeletedAt: null,
	tenant_id: 1,
	patient_id: 7,
	category: "lab_result" as const,
	taken_at: "2026-03-15T00:00:00Z",
	description: "",
	file_name: "hemograma.pdf",
	mime_type: "application/pdf",
	size: 10,
	sha256: "x",
	uploaded_by_id: 1,
};

function renderViewer(attachment: typeof base | null, onClose = vi.fn()) {
	const ui = (a: typeof base | null) => (
		<UiTextProvider value={{ textGet: (key) => key }}>
			<AttachmentViewer
				patientId={7}
				attachment={a}
				onClose={onClose}
				onDownload={vi.fn()}
			/>
		</UiTextProvider>
	);
	const utils = render(ui(attachment));
	return {
		...utils,
		rerenderWith: (a: typeof base | null) => utils.rerender(ui(a)),
	};
}

describe("AttachmentViewer", () => {
	const revoke = vi.fn();
	beforeEach(() => {
		mocks.viewPatientAttachment.mockReset();
		revoke.mockReset();
		window.URL.createObjectURL = vi.fn(() => "blob:file");
		window.URL.revokeObjectURL = revoke;
	});

	it("renders a PDF through the pdf.js pages, never an iframe, with no object URL", async () => {
		mocks.viewPatientAttachment.mockResolvedValue({
			success: true,
			data: new Blob(["x"], { type: "application/pdf" }),
		});
		const { container } = renderViewer(base);
		expect(await screen.findByTestId("pdf-pages")).toHaveTextContent(
			"application/pdf",
		);
		expect(document.querySelector("iframe")).toBeNull();
		expect(container.querySelector("iframe")).toBeNull();
		expect(window.URL.createObjectURL).not.toHaveBeenCalled();
		expect(mocks.viewPatientAttachment).toHaveBeenCalledWith(7, 3);
	});

	it("shows an image with <img> and revokes the URL on close", async () => {
		mocks.viewPatientAttachment.mockResolvedValue({
			success: true,
			data: new Blob(["x"], { type: "image/png" }),
		});
		const { rerenderWith } = renderViewer({
			...base,
			file_name: "rx.png",
			mime_type: "image/png",
		});
		const img = await screen.findByRole("img", { name: "rx.png" });
		expect(img.getAttribute("src")).toBe("blob:file");
		expect(revoke).not.toHaveBeenCalled();
		act(() => rerenderWith(null));
		expect(revoke).toHaveBeenCalledWith("blob:file");
	});

	it("shows an error when the file cannot be loaded", async () => {
		mocks.viewPatientAttachment.mockResolvedValue({ success: false });
		renderViewer(base);
		expect(
			await screen.findByText(/clinical\.attachment\.viewer\.error/),
		).toBeInTheDocument();
		fireEvent.keyDown(document.body, { key: "Escape" });
	});
});
