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
 *   E2E_NOPERM_USERNAME, E2E_NOPERM_PASSWORD (optional)
 *                               a user of the same clinic WITHOUT READ_CASH; the
 *                               permission-denied spec is skipped without them
 * Every test starts by closing any session left open (state reset), so the
 * clinic's desk is closed when they finish. Invoices issued before the first
 * session cannot be charged, which is why the tests create their own.
 */
const username = process.env.E2E_USERNAME;
const password = process.env.E2E_PASSWORD;
const patientName = process.env.E2E_PATIENT_NAME;
const environment = process.env.E2E_ENVIRONMENT;
const noPermUsername = process.env.E2E_NOPERM_USERNAME;
const noPermPassword = process.env.E2E_NOPERM_PASSWORD;

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
	movement: /^(Movimiento|Movement)$/,
	movementIn: /^(Ingreso|Cash in)$/,
	movementOut: /^(Egreso|Cash out)$/,
	amount: /^(Monto|Amount)$/,
	reason: /^(Motivo|Reason)/,
	record: /^(Registrar|Record)$/,
	void: /^(Anular|Void)$/,
	voidSubmit: /^(Anular cobro|Void payment)$/,
	voided: /^(Anulado|Voided)$/,
	noAccess: /^(Acceso denegado|Access denied)$/,
};

async function login(page: Page, user = username, pass = password) {
	await page.goto("/login");
	await page.getByLabel(label.username).fill(user ?? "");
	await page.getByLabel(label.password).fill(pass ?? "");
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

/** Closes the open session (counting 0, with a note) if there is one. */
async function closeDesk(page: Page) {
	await page.getByRole("button", { name: label.closeDesk }).first().click();
	const closeDialog = page.getByRole("dialog");
	await closeDialog.getByLabel(label.counted).fill("0");
	await closeDialog.getByRole("button", { name: label.closeDesk }).click();
	const note = closeDialog.getByLabel(label.note);
	if (await note.isVisible({ timeout: 3000 }).catch(() => false)) {
		await note.fill("Conteo de prueba (e2e)");
		await closeDialog.getByRole("button", { name: label.closeDesk }).click();
	}
	await expect(closeDialog).toBeHidden();
}

/** State reset: leaves the clinic's desk closed, whatever a previous run did. */
async function resetDesk(page: Page) {
	await page.goto("/cash");
	const isOpen = await page
		.getByText(label.deskOpen)
		.first()
		.isVisible({ timeout: 5000 })
		.catch(() => false);
	if (isOpen) {
		await closeDesk(page);
		await page.goto("/cash");
	}
	await expect(page.getByLabel(label.float)).toBeVisible();
}

async function openDesk(page: Page, float: string) {
	await page.getByLabel(label.float).fill(float);
	await page.getByRole("button", { name: label.openDesk }).click();
	await expect(page.getByText(label.deskOpen).first()).toBeVisible();
}

/** Creates a cash invoice; saving opens the charge dialog, which charges it. */
async function createAndChargeInvoice(page: Page) {
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
}

test.describe("Cash desk", () => {
	test.skip(
		!username || !password || !patientName,
		"Set E2E_USERNAME, E2E_PASSWORD and E2E_PATIENT_NAME to run against a real stack",
	);
	// The tests share one clinic and one desk: they cannot run in parallel.
	test.describe.configure({ mode: "serial" });

	test.beforeEach(async ({ page }) => {
		await login(page);
		await resetDesk(page);
	});

	test("open, charge an invoice and close blind", async ({ page }) => {
		await openDesk(page, "20");
		await createAndChargeInvoice(page);

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

	test("void a payment of the open session", async ({ page }) => {
		await openDesk(page, "0");
		await createAndChargeInvoice(page);
		await page.goto("/cash");

		await page.getByRole("button", { name: label.void }).first().click();
		const dialog = page.getByRole("dialog");
		await dialog.getByLabel(label.reason).fill("Cobro duplicado (e2e)");
		await dialog.getByRole("button", { name: label.voidSubmit }).click();
		await expect(dialog).toBeHidden();

		// The payment stays listed, marked as voided, and can't be voided again.
		await expect(page.getByText(label.voided).first()).toBeVisible();
		await expect(page.getByRole("button", { name: label.void })).toHaveCount(0);
	});

	test("record a cash in and a cash out movement", async ({ page }) => {
		await openDesk(page, "0");

		for (const [type, amount, reason] of [
			[label.movementIn, "15", "Cambio de caja (e2e)"],
			[label.movementOut, "5", "Retiro a banco (e2e)"],
		] as const) {
			await page.getByRole("button", { name: label.movement }).click();
			const dialog = page.getByRole("dialog");
			await dialog.getByRole("radio", { name: type }).check({ force: true });
			await dialog.getByLabel(label.amount).fill(amount);
			await dialog.getByLabel(label.reason).fill(reason);
			await dialog.getByRole("button", { name: label.record }).click();
			await expect(dialog).toBeHidden();
			await expect(page.getByText(reason)).toBeVisible();
		}
	});
});

test.describe("Cash desk permissions", () => {
	test.skip(
		!noPermUsername || !noPermPassword,
		"Set E2E_NOPERM_USERNAME and E2E_NOPERM_PASSWORD (a user without READ_CASH)",
	);

	test("a role without cash permission cannot reach the page", async ({
		page,
	}) => {
		await login(page, noPermUsername, noPermPassword);
		await page.goto("/cash");
		// The route guard shows the no-access screen instead of the desk.
		await expect(page.getByText(label.noAccess)).toBeVisible();
		await expect(page.getByLabel(label.float)).toHaveCount(0);
		await expect(
			page.getByRole("button", { name: label.openDesk }),
		).toHaveCount(0);
		await expect(page.getByText(label.deskOpen)).toHaveCount(0);
	});
});
