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
	updatePatientAttachment: vi.fn(),
	downloadPatientAttachment: vi.fn(),
	deletePatientAttachment: vi.fn(),
	getDeletedPatientAttachments: vi.fn(),
	restorePatientAttachment: vi.fn(),
}));

vi.mock("@/api/patient-attachment-service", () => ({
	...mocks,
	ATTACHMENT_CATEGORIES: [
		"lab_result",
		"imaging",
		"external_report",
		"clinical_photo",
		"consent",
		"other",
	],
	MAX_ATTACHMENT_SIZE: 15 * 1024 * 1024,
	isStorageFull: (u: { used_bytes: number; quota_bytes: number }) =>
		u.used_bytes >= u.quota_bytes,
}));

import { PatientAttachmentsPanel } from "@/components/features/patient-attachments/patient-attachments-panel";

function renderPanel(props: {
	canRead: boolean;
	canUpload: boolean;
	canDelete?: boolean;
}) {
	return render(
		<UiTextProvider value={{ textGet: (key) => key }}>
			<PatientAttachmentsPanel patientId={7} {...props} />
		</UiTextProvider>,
	);
}

const attachment = {
	ID: 3,
	CreatedAt: "2026-10-02T10:00:00Z",
	UpdatedAt: "2026-10-02T10:00:00Z",
	DeletedAt: null,
	tenant_id: 1,
	patient_id: 7,
	category: "lab_result",
	taken_at: "2026-03-15T00:00:00Z",
	description: "Hemograma Lab. X",
	file_name: "hemograma.pdf",
	mime_type: "application/pdf",
	size: 2048,
	sha256: "abc",
	uploaded_by_id: 1,
};

const GB = 1024 ** 3;

/** A list response: the items plus the tenant's storage usage. */
function listed(
	items: unknown[],
	usage = { used_bytes: 0, quota_bytes: 10 * GB, warning: false },
) {
	return { success: true, data: { items, usage } };
}

describe("PatientAttachmentsPanel", () => {
	beforeEach(() => {
		act(() => {
			useMessageStore.setState({ lang: "es", messages: {} });
		});
		for (const fn of Object.values(mocks)) fn.mockReset();
	});

	it("explains what can be uploaded when the patient has no files", async () => {
		mocks.getPatientAttachments.mockResolvedValue(listed([]));
		renderPanel({ canRead: true, canUpload: true });

		expect(
			await screen.findByText(/clinical\.attachment\.empty\.title/),
		).toBeInTheDocument();
		expect(mocks.getPatientAttachments).toHaveBeenCalledWith(7, undefined);
		expect(
			screen.getByRole("button", { name: /clinical\.attachment\.upload/ }),
		).toBeInTheDocument();
	});

	it("lists the files and downloads the chosen one", async () => {
		mocks.getPatientAttachments.mockResolvedValue(listed([attachment]));
		mocks.downloadPatientAttachment.mockResolvedValue({ success: false });
		renderPanel({ canRead: true, canUpload: false });

		expect(await screen.findByText("hemograma.pdf")).toBeInTheDocument();
		expect(screen.getByText("Hemograma Lab. X")).toBeInTheDocument();
		expect(
			screen.queryByRole("button", { name: /clinical\.attachment\.upload/ }),
		).not.toBeInTheDocument();

		fireEvent.click(
			screen.getByRole("button", { name: /clinical\.attachment\.download/ }),
		);
		expect(mocks.downloadPatientAttachment).toHaveBeenCalledWith(7, 3);
	});

	it("lets a user who can only upload do so without listing files", () => {
		renderPanel({ canRead: false, canUpload: true });

		expect(
			screen.getByText(/clinical\.attachment\.no_read/),
		).toBeInTheDocument();
		expect(
			screen.getByRole("button", { name: /clinical\.attachment\.upload/ }),
		).toBeInTheDocument();
		expect(mocks.getPatientAttachments).not.toHaveBeenCalled();
	});

	it("edits the metadata of a file, prefilled, without a file picker", async () => {
		mocks.getPatientAttachments.mockResolvedValue(listed([attachment]));
		mocks.updatePatientAttachment.mockResolvedValue({ success: true });
		renderPanel({ canRead: true, canUpload: true });

		fireEvent.click(
			await screen.findByRole("button", { name: /clinical\.attachment\.edit/ }),
		);
		expect(
			await screen.findByDisplayValue("Hemograma Lab. X"),
		).toBeInTheDocument();
		expect(document.getElementById("attachment-file")).toBeNull();

		fireEvent.change(screen.getByDisplayValue("Hemograma Lab. X"), {
			target: { value: "Hemograma control" },
		});
		fireEvent.click(
			screen.getByRole("button", { name: /clinical\.attachment\.form\.save/ }),
		);

		await waitFor(() =>
			expect(mocks.updatePatientAttachment).toHaveBeenCalledWith(7, 3, {
				category: "lab_result",
				taken_at: "2026-03-15",
				description: "Hemograma control",
			}),
		);
		await waitFor(() =>
			expect(mocks.getPatientAttachments).toHaveBeenCalledTimes(2),
		);
	});

	it("hides the edit action from users who cannot upload", async () => {
		mocks.getPatientAttachments.mockResolvedValue(listed([attachment]));
		renderPanel({ canRead: true, canUpload: false });

		await screen.findByText("hemograma.pdf");
		expect(
			screen.queryByRole("button", { name: /clinical\.attachment\.edit/ }),
		).not.toBeInTheDocument();
	});

	it("requires a reason before deleting, then calls the service and reloads", async () => {
		mocks.getPatientAttachments.mockResolvedValue(listed([attachment]));
		mocks.deletePatientAttachment.mockResolvedValue({ success: true });
		renderPanel({ canRead: true, canUpload: false, canDelete: true });

		fireEvent.click(
			await screen.findByRole("button", {
				name: /clinical\.attachment\.delete\*$/,
			}),
		);
		fireEvent.click(
			await screen.findByRole("button", {
				name: /clinical\.attachment\.delete\.confirm/,
			}),
		);
		expect(
			await screen.findByText(
				/clinical\.attachment\.delete\.error\.reason_required/,
			),
		).toBeInTheDocument();
		expect(mocks.deletePatientAttachment).not.toHaveBeenCalled();

		fireEvent.change(
			screen.getByPlaceholderText(
				/clinical\.attachment\.delete\.reason\.placeholder/,
			),
			{ target: { value: "  Subido al paciente equivocado  " } },
		);
		fireEvent.click(
			screen.getByRole("button", {
				name: /clinical\.attachment\.delete\.confirm/,
			}),
		);

		await waitFor(() =>
			expect(mocks.deletePatientAttachment).toHaveBeenCalledWith(
				7,
				3,
				"Subido al paciente equivocado",
			),
		);
		await waitFor(() =>
			expect(mocks.getPatientAttachments).toHaveBeenCalledTimes(2),
		);
	});

	it("lists deleted files with who and why, and restores one", async () => {
		mocks.getPatientAttachments.mockResolvedValue(listed([]));
		mocks.getDeletedPatientAttachments.mockResolvedValue({
			success: true,
			data: [
				{
					...attachment,
					DeletedAt: "2026-10-03T15:00:00Z",
					deleted_by_id: 9,
					deleted_by_name: "dra.perez",
					delete_reason: "Paciente equivocado",
				},
			],
		});
		mocks.restorePatientAttachment.mockResolvedValue({ success: true });
		renderPanel({ canRead: true, canUpload: false, canDelete: true });

		fireEvent.click(
			await screen.findByRole("button", {
				name: /clinical\.attachment\.deleted\.show/,
			}),
		);
		expect(await screen.findByText("dra.perez")).toBeInTheDocument();
		expect(screen.getByText("Paciente equivocado")).toBeInTheDocument();
		expect(mocks.getDeletedPatientAttachments).toHaveBeenCalledWith(7);

		fireEvent.click(
			screen.getByRole("button", { name: /clinical\.attachment\.restore/ }),
		);
		await waitFor(() =>
			expect(mocks.restorePatientAttachment).toHaveBeenCalledWith(7, 3),
		);
		await waitFor(() =>
			expect(mocks.getDeletedPatientAttachments).toHaveBeenCalledTimes(2),
		);
	});

	it("hides delete and the deleted view from users without the permission", async () => {
		mocks.getPatientAttachments.mockResolvedValue(listed([attachment]));
		renderPanel({ canRead: true, canUpload: true, canDelete: false });

		await screen.findByText("hemograma.pdf");
		expect(
			screen.queryByRole("button", { name: /clinical\.attachment\.delete\*$/ }),
		).not.toBeInTheDocument();
		expect(
			screen.queryByRole("button", {
				name: /clinical\.attachment\.deleted\.show/,
			}),
		).not.toBeInTheDocument();
	});

	it("disables delete for the result of a reviewed exam", async () => {
		mocks.getPatientAttachments.mockResolvedValue(
			listed([{ ...attachment, locked_by_review: true }]),
		);
		renderPanel({ canRead: true, canUpload: true, canDelete: true });

		await screen.findByText("hemograma.pdf");
		expect(
			screen.queryByRole("button", { name: /clinical\.attachment\.delete\*$/ }),
		).not.toBeInTheDocument();
		expect(
			screen.getByRole("button", {
				name: /clinical\.attachment\.delete\.locked_by_review/,
			}),
		).toBeDisabled();
	});

	describe("storage usage", () => {
		const uploadButton = () =>
			screen.getByRole("button", { name: /clinical\.attachment\.upload/ });

		beforeEach(() => {
			act(() => {
				useMessageStore.setState({
					lang: "es",
					messages: { "clinical.attachment.usage.amount": "{used} de {quota}" },
				});
			});
		});

		it("shows the bar with what is used of the quota", async () => {
			mocks.getPatientAttachments.mockResolvedValue(
				listed([attachment], {
					used_bytes: 1 * GB,
					quota_bytes: 10 * GB,
					warning: false,
				}),
			);
			renderPanel({ canRead: true, canUpload: true });

			const bar = await screen.findByRole("progressbar");
			expect(bar).toHaveAttribute("aria-valuenow", "10");
			expect(screen.getByText(/^1\sGB de 10\sGB$/)).toBeInTheDocument();
			expect(
				screen.queryByText(/clinical\.attachment\.usage\.warning/),
			).not.toBeInTheDocument();
			expect(uploadButton()).toBeEnabled();
		});

		it("warns from 80% and still lets upload", async () => {
			mocks.getPatientAttachments.mockResolvedValue(
				listed([attachment], {
					used_bytes: 8 * GB,
					quota_bytes: 10 * GB,
					warning: true,
				}),
			);
			renderPanel({ canRead: true, canUpload: true });

			expect(
				await screen.findByText(/clinical\.attachment\.usage\.warning/),
			).toBeInTheDocument();
			expect(screen.getByRole("progressbar")).toHaveAttribute(
				"aria-valuenow",
				"80",
			);
			expect(uploadButton()).toBeEnabled();
		});

		it("disables upload with a message once the quota is full", async () => {
			mocks.getPatientAttachments.mockResolvedValue(
				listed([attachment], {
					used_bytes: 10 * GB,
					quota_bytes: 10 * GB,
					warning: true,
				}),
			);
			renderPanel({ canRead: true, canUpload: true });

			expect(
				await screen.findByText(/clinical\.attachment\.usage\.full/),
			).toBeInTheDocument();
			expect(
				screen.queryByText(/clinical\.attachment\.usage\.warning/),
			).not.toBeInTheDocument();
			expect(uploadButton()).toBeDisabled();
		});

		it("disables upload when the plan includes no storage", async () => {
			mocks.getPatientAttachments.mockResolvedValue(
				listed([], { used_bytes: 0, quota_bytes: 0, warning: true }),
			);
			renderPanel({ canRead: true, canUpload: true });

			expect(
				await screen.findByText(/clinical\.attachment\.usage\.not_included/),
			).toBeInTheDocument();
			expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
			expect(uploadButton()).toBeDisabled();
		});
	});

	describe("multi-file upload", () => {
		it("uploads each file with its own state and warns about a duplicate", async () => {
			mocks.getPatientAttachments.mockResolvedValue(listed([]));
			mocks.uploadPatientAttachment
				.mockResolvedValueOnce({ success: true, data: { ...attachment } })
				.mockResolvedValueOnce({
					success: true,
					data: {
						...attachment,
						duplicate_of: {
							id: 3,
							file_name: "hemograma.pdf",
							created_at: "2026-10-01T10:00:00Z",
						},
					},
				});
			renderPanel({ canRead: true, canUpload: true });

			fireEvent.click(
				await screen.findByRole("button", {
					name: /clinical\.attachment\.upload/,
				}),
			);
			const input = await screen.findByTestId("attachment-file-input");
			expect(input).toHaveAttribute("multiple");
			expect(input.getAttribute("accept")).toContain("image/heic");
			expect(screen.getByTestId("attachment-camera-input")).toHaveAttribute(
				"capture",
				"environment",
			);
			fireEvent.change(input, {
				target: {
					files: [
						new File(["a"], "a.pdf", { type: "application/pdf" }),
						new File(["b"], "b.pdf", { type: "application/pdf" }),
					],
				},
			});
			// Category: the form's select.
			fireEvent.click(screen.getAllByRole("combobox").at(-1) as HTMLElement);
			fireEvent.click(
				await screen.findByRole("option", { name: /category\.lab_result/ }),
			);
			fireEvent.click(
				screen.getByRole("button", {
					name: /clinical\.attachment\.form\.submit/,
				}),
			);

			await waitFor(() =>
				expect(mocks.uploadPatientAttachment).toHaveBeenCalledTimes(2),
			);
			expect(mocks.uploadPatientAttachment.mock.calls[0][1].file.name).toBe(
				"a.pdf",
			);
			expect(
				await screen.findByText(
					/clinical\.attachment\.upload\.status\.duplicate/,
				),
			).toBeInTheDocument();
			expect(
				screen.getByText(/clinical\.attachment\.upload\.status\.done/),
			).toBeInTheDocument();
		});
	});
});
