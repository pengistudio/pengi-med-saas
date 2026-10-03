import { expect, type Page, test } from "@playwright/test";

/**
 * Patient attachments (Adjuntos), end to end against a running stack: upload a
 * PDF from the patient's "Archivos" tab, see it listed, open the viewer,
 * delete it with a reason, find it under "Eliminados" and restore it.
 *
 * Needs a real user and patient, given through the environment:
 *   E2E_USERNAME, E2E_PASSWORD  a user whose role and plan include
 *                               READ/UPLOAD/DELETE_PATIENT_ATTACHMENT
 *   E2E_PATIENT_ID              a patient of that user's clinic
 *   E2E_ENVIRONMENT (optional)  clinic name to pick when the user has several
 * and an API started with ATTACHMENT_ENCRYPTION_KEY (otherwise every attachment
 * route answers 503).
 */
const username = process.env.E2E_USERNAME;
const password = process.env.E2E_PASSWORD;
const patientId = process.env.E2E_PATIENT_ID;
const environment = process.env.E2E_ENVIRONMENT;

// A minimal valid PDF; the server decides the type from these bytes.
const samplePdf = Buffer.from(
	"%PDF-1.4\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n" +
		"2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n" +
		"3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >> endobj\n" +
		"trailer << /Root 1 0 R >>\n%%EOF\n",
);

// UI labels in either interface language (es / en).
const label = {
	username: /Nombre de usuario|Username/,
	password: /Contraseña|Password/,
	login: /Iniciar sesión|Login/,
	filesTab: /^(Archivos|Files)$/,
	upload: /^(Subir archivo|Upload file)$/,
	submit: /^(Subir|Upload)$/,
	labResult: /Resultado de laboratorio|Lab result/,
	close: /^(Cerrar|Close)$/,
	view: /^(Ver|View)$/,
	delete: /^(Eliminar|Delete)$/,
	deleteReason: /Motivo de la eliminación|Reason for deleting/,
	deleteConfirm: /Eliminar adjunto|Delete attachment/,
	showDeleted: /^(Eliminados|Deleted)$/,
	restore: /^(Restaurar|Restore)$/,
};

async function login(page: Page) {
	await page.goto("/login");
	await page.getByLabel(label.username).fill(username ?? "");
	await page.getByLabel(label.password).fill(password ?? "");
	await page.getByRole("button", { name: label.login }).click();
	// One clinic is picked automatically; with several, choose one.
	await page.waitForURL(/\/login\/environments|\/$|\/dashboard/);
	if (page.url().includes("/login/environments")) {
		const choice = environment
			? page.getByRole("button", { name: new RegExp(environment) })
			: page.locator("button[type=button]").first();
		if (await choice.isVisible({ timeout: 3000 }).catch(() => false)) {
			await choice.click();
		}
		await page.waitForURL((url) => !url.pathname.startsWith("/login"));
	}
}

test.describe("Patient attachments", () => {
	test.skip(
		!username || !password || !patientId,
		"Set E2E_USERNAME, E2E_PASSWORD and E2E_PATIENT_ID to run against a real stack",
	);

	test("upload a PDF, view it, delete it with a reason and restore it", async ({
		page,
	}) => {
		const fileName = `e2e-${Date.now()}.pdf`;
		const reason = "Subido por error (e2e)";

		await login(page);
		await page.goto(`/clinical/medical-documents?patient_id=${patientId}`);
		await page.getByRole("tab", { name: label.filesTab }).click();

		// Upload
		await page.getByRole("button", { name: label.upload }).click();
		const dialog = page.getByRole("dialog");
		await dialog.getByTestId("attachment-file-input").setInputFiles({
			name: fileName,
			mimeType: "application/pdf",
			buffer: samplePdf,
		});
		await dialog.getByRole("combobox").last().click();
		await page.getByRole("option", { name: label.labResult }).click();
		await dialog.getByRole("button", { name: label.submit }).click();
		await expect(dialog.getByText(fileName)).toBeVisible();
		await dialog.getByRole("button", { name: label.close }).click();

		// Listed
		const row = page.getByRole("row").filter({ hasText: fileName });
		await expect(row).toBeVisible();

		// Viewer: the PDF is drawn on a canvas inside the dialog.
		await row.getByRole("button", { name: label.view }).click();
		const viewer = page.getByRole("dialog");
		await expect(viewer.getByText(fileName)).toBeVisible();
		await expect(viewer.locator("canvas").first()).toBeVisible({
			timeout: 15000,
		});
		await page.keyboard.press("Escape");
		await expect(viewer).toBeHidden();

		// Delete with a reason
		await row.getByRole("button", { name: label.delete }).click();
		const deleteDialog = page.getByRole("dialog");
		await deleteDialog.getByLabel(label.deleteReason).fill(reason);
		await deleteDialog
			.getByRole("button", { name: label.deleteConfirm })
			.click();
		await expect(deleteDialog).toBeHidden();
		await expect(
			page.getByRole("row").filter({ hasText: fileName }),
		).toHaveCount(0);

		// Listed under "Eliminados", with the reason
		await page.getByRole("button", { name: label.showDeleted }).click();
		const deletedRow = page.getByRole("row").filter({ hasText: fileName });
		await expect(deletedRow).toBeVisible();
		await expect(deletedRow).toContainText(reason);

		// Restore: gone from the deleted list, back in the normal one.
		await deletedRow.getByRole("button", { name: label.restore }).click();
		await expect(deletedRow).toHaveCount(0);
		await page
			.getByRole("button", { name: /Ver adjuntos|Back to attachments/ })
			.click();
		await expect(
			page.getByRole("row").filter({ hasText: fileName }),
		).toBeVisible();
	});
});
