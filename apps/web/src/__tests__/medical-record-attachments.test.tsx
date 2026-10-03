import { useMessageStore } from "@pengi/shared";
import { UiTextProvider } from "@pengi/ui";
import {
	act,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
	getPatientAttachments: vi.fn(),
	uploadPatientAttachment: vi.fn(),
	linkPatientAttachment: vi.fn(),
	downloadPatientAttachment: vi.fn(),
	viewPatientAttachment: vi.fn(),
	permissions: new Set<string>(),
}));

vi.mock("@/api/patient-attachment-service", () => ({
	getPatientAttachments: mocks.getPatientAttachments,
	uploadPatientAttachment: mocks.uploadPatientAttachment,
	linkPatientAttachment: mocks.linkPatientAttachment,
	downloadPatientAttachment: mocks.downloadPatientAttachment,
	viewPatientAttachment: mocks.viewPatientAttachment,
	ATTACHMENT_CATEGORIES: [
		"lab_result",
		"imaging",
		"external_report",
		"clinical_photo",
		"consent",
		"other",
	],
	MAX_ATTACHMENT_SIZE: 15 * 1024 * 1024,
	STORAGE_QUOTA_ERROR_CODE: "E-PLAN-003",
}));

vi.mock("@/hooks/use-permission", () => ({
	default: () => ({
		checkPermission: (required: string[]) =>
			required.every((p) => mocks.permissions.has(p)),
	}),
}));

import type { PatientAttachment } from "@/api/patient-attachment-service";
import { MedicalRecordAttachments } from "@/components/features/patient-attachments/medical-record-attachments";

const READ = "READ_PATIENT_ATTACHMENT";
const UPLOAD = "UPLOAD_PATIENT_ATTACHMENT";

function attachment(
	ID: number,
	file_name: string,
	medical_record_id: number | null = null,
): PatientAttachment {
	return {
		ID,
		CreatedAt: "2026-10-02T10:00:00Z",
		UpdatedAt: "2026-10-02T10:00:00Z",
		DeletedAt: null,
		tenant_id: 1,
		patient_id: 7,
		medical_record_id,
		category: "lab_result",
		taken_at: "2026-03-15T00:00:00Z",
		description: "",
		file_name,
		mime_type: "application/pdf",
		size: 2048,
		sha256: "abc",
		uploaded_by_id: 1,
	} as PatientAttachment;
}

const listed = (items: PatientAttachment[]) => ({
	success: true,
	data: {
		items,
		usage: { used_bytes: 0, quota_bytes: 1024 ** 3, warning: false },
	},
});

const outerSubmit = vi.fn((e: React.FormEvent) => e.preventDefault());

function renderSection(props: {
	medicalRecordId?: number;
	pendingAttachments?: PatientAttachment[];
	onPendingChange?: (a: PatientAttachment[]) => void;
}) {
	// Inside a form, like the consultation form it lives in.
	return render(
		<UiTextProvider value={{ textGet: (key) => key }}>
			<form onSubmit={outerSubmit}>
				<MedicalRecordAttachments patientId={7} {...props} />
			</form>
		</UiTextProvider>,
	);
}

async function uploadOneFile() {
	fireEvent.click(
		screen.getByRole("button", { name: /\*clinical\.attachment\.upload\*/ }),
	);
	const input = await screen.findByTestId("attachment-file-input");
	fireEvent.change(input, {
		target: {
			files: [new File(["a"], "a.pdf", { type: "application/pdf" })],
		},
	});
	fireEvent.click(screen.getAllByRole("combobox").at(-1) as HTMLElement);
	fireEvent.click(
		await screen.findByRole("option", { name: /category\.lab_result/ }),
	);
	fireEvent.click(
		screen.getByRole("button", { name: /clinical\.attachment\.form\.submit/ }),
	);
	await waitFor(() =>
		expect(mocks.uploadPatientAttachment).toHaveBeenCalledTimes(1),
	);
}

describe("MedicalRecordAttachments", () => {
	beforeEach(() => {
		act(() => {
			useMessageStore.setState({ lang: "es", messages: {} });
		});
		mocks.getPatientAttachments.mockReset();
		mocks.uploadPatientAttachment.mockReset();
		mocks.linkPatientAttachment.mockReset();
		outerSubmit.mockClear();
		mocks.permissions.clear();
		mocks.permissions.add(READ);
		mocks.permissions.add(UPLOAD);
	});

	describe("on a saved consultation", () => {
		it("lists only the files linked to it", async () => {
			mocks.getPatientAttachments.mockResolvedValue(
				listed([attachment(3, "hemograma.pdf", 12)]),
			);
			renderSection({ medicalRecordId: 12 });

			expect(await screen.findByText("hemograma.pdf")).toBeInTheDocument();
			expect(mocks.getPatientAttachments).toHaveBeenCalledWith(
				7,
				undefined,
				12,
			);
		});

		it("uploads linked to the consultation without submitting the consultation form", async () => {
			mocks.getPatientAttachments.mockResolvedValue(listed([]));
			mocks.uploadPatientAttachment.mockResolvedValue({
				success: true,
				data: attachment(4, "a.pdf", 12),
			});
			renderSection({ medicalRecordId: 12 });
			await screen.findByText(/clinical\.attachment\.consultation\.empty/);

			await uploadOneFile();

			expect(mocks.uploadPatientAttachment.mock.calls[0][1]).toMatchObject({
				medical_record_id: 12,
				category: "lab_result",
			});
			expect(outerSubmit).not.toHaveBeenCalled();
			// Reloaded after the upload.
			await waitFor(() =>
				expect(mocks.getPatientAttachments).toHaveBeenCalledTimes(2),
			);
		});

		it("unlinks a file", async () => {
			mocks.getPatientAttachments.mockResolvedValue(
				listed([attachment(3, "hemograma.pdf", 12)]),
			);
			mocks.linkPatientAttachment.mockResolvedValue({ success: true });
			renderSection({ medicalRecordId: 12 });

			fireEvent.click(
				await screen.findByRole("button", {
					name: /clinical\.attachment\.consultation\.unlink/,
				}),
			);
			await waitFor(() =>
				expect(mocks.linkPatientAttachment).toHaveBeenCalledWith(7, 3, null),
			);
		});

		it("links an existing file, offering only the patient's unlinked ones", async () => {
			mocks.getPatientAttachments.mockImplementation(
				async (_p: number, _c?: string, recordId?: number) =>
					recordId
						? listed([])
						: listed([
								attachment(5, "rx-torax.png"),
								attachment(6, "otra-consulta.pdf", 99),
							]),
			);
			mocks.linkPatientAttachment.mockResolvedValue({ success: true });
			renderSection({ medicalRecordId: 12 });
			await screen.findByText(/clinical\.attachment\.consultation\.empty/);

			fireEvent.click(
				screen.getByRole("button", {
					name: /clinical\.attachment\.consultation\.link_existing/,
				}),
			);
			expect(await screen.findByText("rx-torax.png")).toBeInTheDocument();
			expect(screen.queryByText("otra-consulta.pdf")).not.toBeInTheDocument();

			fireEvent.click(
				screen.getByRole("button", {
					name: /\*clinical\.attachment\.consultation\.link\*/,
				}),
			);
			await waitFor(() =>
				expect(mocks.linkPatientAttachment).toHaveBeenCalledWith(7, 5, 12),
			);
		});
	});

	describe("on a consultation being written", () => {
		it("uploads unlinked and hands the files to the form as pending", async () => {
			mocks.uploadPatientAttachment.mockResolvedValue({
				success: true,
				data: attachment(4, "a.pdf"),
			});
			const onPendingChange = vi.fn();
			renderSection({ pendingAttachments: [], onPendingChange });

			// Nothing to load: the consultation doesn't exist yet.
			expect(mocks.getPatientAttachments).not.toHaveBeenCalled();
			await uploadOneFile();

			expect(
				mocks.uploadPatientAttachment.mock.calls[0][1].medical_record_id,
			).toBeUndefined();
			await waitFor(() =>
				expect(onPendingChange).toHaveBeenCalledWith([
					expect.objectContaining({ ID: 4 }),
				]),
			);
			expect(outerSubmit).not.toHaveBeenCalled();
		});

		it("drops a pending file without calling the API", () => {
			const onPendingChange = vi.fn();
			renderSection({
				pendingAttachments: [attachment(4, "a.pdf"), attachment(5, "b.pdf")],
				onPendingChange,
			});

			fireEvent.click(
				screen.getAllByRole("button", {
					name: /clinical\.attachment\.consultation\.unlink/,
				})[0],
			);
			expect(onPendingChange).toHaveBeenCalledWith([
				expect.objectContaining({ ID: 5 }),
			]);
			expect(mocks.linkPatientAttachment).not.toHaveBeenCalled();
		});
	});

	describe("permissions", () => {
		it("renders nothing without attachment permissions", () => {
			mocks.permissions.clear();
			const { container } = renderSection({ medicalRecordId: 12 });
			expect(container.querySelector("section")).toBeNull();
			expect(mocks.getPatientAttachments).not.toHaveBeenCalled();
		});

		it("only lists and views with read permission", async () => {
			mocks.permissions.delete(UPLOAD);
			mocks.getPatientAttachments.mockResolvedValue(
				listed([attachment(3, "hemograma.pdf", 12)]),
			);
			renderSection({ medicalRecordId: 12 });

			expect(await screen.findByText("hemograma.pdf")).toBeInTheDocument();
			expect(
				screen.getByRole("button", { name: /clinical\.attachment\.view/ }),
			).toBeInTheDocument();
			expect(
				screen.queryByRole("button", { name: /clinical\.attachment\.upload/ }),
			).not.toBeInTheDocument();
			expect(
				screen.queryByRole("button", {
					name: /clinical\.attachment\.consultation\.(unlink|link_existing)/,
				}),
			).not.toBeInTheDocument();
		});

		it("lets an upload-only user upload without listing", () => {
			mocks.permissions.delete(READ);
			renderSection({ medicalRecordId: 12 });

			expect(
				screen.getByText(/clinical\.attachment\.no_read/),
			).toBeInTheDocument();
			expect(
				screen.getByRole("button", {
					name: /\*clinical\.attachment\.upload\*/,
				}),
			).toBeInTheDocument();
			expect(mocks.getPatientAttachments).not.toHaveBeenCalled();
		});
	});
});
