import { expect, type Page, test } from "@playwright/test";

/**
 * Cash desk (Caja), end to end against a running stack: open the desk with a
 * float, create an invoice (saving it opens the charge dialog with the balance
 * prefilled, paid in cash), then close blind: the expected amount is not shown
 * while counting and appears only in the closing summary.
 *
 * Needs a real user and a clinic with the CASH feature, through the environment:
 *   E2E_USERNAME, E2E_PASSWORD  a user whose role and plan include READ_CASH,
 *                               CREATE_PAYMENT and MANAGE_CASH_SESSION
 *   E2E_PATIENT_NAME            text that finds one patient of the clinic in the
 *                               invoice form (e.g. a first name)
 *   E2E_ENVIRONMENT (optional)  clinic name to pick when the user has several
 * The desk must be closed: an already open session is left alone and the test
 * is skipped. Invoices issued before the first session cannot be charged, which
 * is why the test creates its own.
 */
const username = process.env.E2E_USERNAME;
const password = process.env.E2E_PASSWORD;
const patientName = process.env.E2E_PATIENT_NAME;
const environment = process.env.E2E_ENVIRONMENT;

// UI labels in either interface language (es / en).
const label = {
	username: /Usuario o correo|Username or email/,
	password: /Contraseña|Password/,
	deskOpen: /^(Caja abierta|Cash desk open)$/,
	float: /Fondo inicial|Opening float/,
	openDesk: /^(Abrir caja|Open cash desk)$/,
	charge: /^(Cobrar|Charge)$/,
	saveInvoice: /^(Guardar Factura|Save Invoice)$/,
	patient: /Seleccionar paciente|Select patient/,
	method: /Método de Pago|Payment method/,
	noFinancial:
		/Sin utilización del sistema financiero|Without use of the financial system/,
	description: /Escriba o seleccione|Type or select/,
	consult: /Consulta Medica/,
	closeDesk: /^(Cerrar caja|Close cash desk)$/,
	counted: /Efectivo contado|Counted cash/,
	note: /^(Nota|Note)/,
	summary: /Cierre de caja|Cash closing/,
	expected: /Esperado|Expected/,
};

async function login(page: Page) {
	await page.goto("/login");
	await page.getByLabel(label.username).fill(username ?? "");
	await page.getByLabel(label.password).fill(password ?? "");
	await page.locator("button[type=submit]").click();
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

test.describe("Cash desk", () => {
	test.skip(
		!username || !password || !patientName,
		"Set E2E_USERNAME, E2E_PASSWORD and E2E_PATIENT_NAME to run against a real stack",
	);

	test("open, charge an invoice and close blind", async ({ page }) => {
		await login(page);
		await page.goto("/cash");

		const openForm = page.getByLabel(label.float);
		const alreadyOpen = await page
			.getByText(label.deskOpen)
			.first()
			.isVisible({ timeout: 5000 })
			.catch(() => false);
		test.skip(alreadyOpen, "A cash session is already open for this clinic");

		// Open with a float
		await openForm.fill("20");
		await page.getByRole("button", { name: label.openDesk }).click();
		await expect(page.getByText(label.deskOpen).first()).toBeVisible();

		// Create an invoice for the patient: saving opens the charge dialog.
		await page.goto("/billing/create");
		await page.getByRole("combobox", { name: label.patient }).click();
		await page.keyboard.type(patientName ?? "");
		await page.getByRole("option").first().click();
		await page.getByRole("combobox", { name: label.method }).click();
		await page.getByRole("option", { name: label.noFinancial }).click();
		await page.getByRole("combobox", { name: label.description }).click();
		await page.getByRole("option", { name: label.consult }).click();
		await page.getByRole("button", { name: label.saveInvoice }).click();

		// The balance is prefilled and the method is cash: charge it.
		const paymentDialog = page.getByRole("dialog");
		await expect(paymentDialog).toBeVisible({ timeout: 15000 });
		await paymentDialog
			.getByRole("button", { name: label.charge })
			.last()
			.click();
		await expect(paymentDialog).toBeHidden();

		// Blind close: the expected amount is not shown while counting.
		await page.goto("/cash");
		await page.getByRole("button", { name: label.closeDesk }).first().click();
		const closeDialog = page.getByRole("dialog");
		await expect(closeDialog.getByText(label.expected)).toHaveCount(0);
		await closeDialog.getByLabel(label.counted).fill("0");
		await closeDialog.getByRole("button", { name: label.closeDesk }).click();

		// A count that does not match asks for a note, then closes.
		const note = closeDialog.getByLabel(label.note);
		if (await note.isVisible({ timeout: 3000 }).catch(() => false)) {
			await note.fill("Conteo de prueba (e2e)");
			await closeDialog.getByRole("button", { name: label.closeDesk }).click();
		}
		await expect(closeDialog).toBeHidden();

		// Only now is the expected amount shown, in the closing summary.
		await expect(page.getByText(label.summary).first()).toBeVisible();
		await expect(page.getByText(label.expected).first()).toBeVisible();
	});
});
