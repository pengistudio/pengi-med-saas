import { expect, type Page, test } from "@playwright/test";

/**
 * Reports, end to end against a running stack: the page loads for a user with
 * READ_REPORTS, the tabs switch, the CSV export downloads a file, and a user
 * without the permission is denied.
 *
 * Needs real users, through the environment:
 *   E2E_USERNAME, E2E_PASSWORD  a user whose role and plan include READ_REPORTS
 *   E2E_ENVIRONMENT (optional)  clinic name to pick when the user has several
 *   E2E_NOPERM_USERNAME, E2E_NOPERM_PASSWORD (optional)
 *                               a user of the same clinic without READ_REPORTS
 *                               (or a plan without it); the denial test is
 *                               skipped when unset
 */
const username = process.env.E2E_USERNAME;
const password = process.env.E2E_PASSWORD;
const noPermUsername = process.env.E2E_NOPERM_USERNAME;
const noPermPassword = process.env.E2E_NOPERM_PASSWORD;
const environment = process.env.E2E_ENVIRONMENT;

// UI labels in either interface language (es / en).
const label = {
	username: /Usuario o correo|Username or email/,
	password: /Contraseña|Password/,
	title: /Reportes y estadísticas|Reports and statistics/,
	csv: /Descargar CSV|Download CSV/,
	denied: /Acceso denegado|Access denied/,
	planLocked: /Esta función no está en tu plan|This feature isn't in your plan/,
	tabs: {
		revenue: /^(Ingresos|Revenue)$/,
		receivables: /^(Por cobrar|Receivables)$/,
		appointments: /^(Citas|Appointments)$/,
		clinical: /^(Clínico|Clinical)$/,
	},
};

async function login(page: Page, user: string, pass: string) {
	await page.goto("/login");
	await page.getByLabel(label.username).fill(user);
	await page.getByLabel(label.password).fill(pass);
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

test.describe("Reports", () => {
	test.skip(
		!username || !password,
		"Set E2E_USERNAME and E2E_PASSWORD to run against a real stack",
	);

	test("loads, switches tabs and exports CSV", async ({ page }) => {
		await login(page, username ?? "", password ?? "");
		await page.goto("/reports");
		await expect(page.getByText(label.title).first()).toBeVisible();

		const tabs = page.getByRole("tab");
		await expect(tabs).toHaveCount(4);
		await expect(
			page.getByRole("tab", { name: label.tabs.revenue }),
		).toHaveAttribute("aria-selected", "true");

		for (const name of [
			label.tabs.receivables,
			label.tabs.appointments,
			label.tabs.clinical,
			label.tabs.revenue,
		]) {
			const tab = page.getByRole("tab", { name });
			await tab.click();
			await expect(tab).toHaveAttribute("aria-selected", "true");
			// The report settles into data or the empty state, never an error.
			await expect(page.locator("[aria-busy=true]")).toHaveCount(0, {
				timeout: 15000,
			});
			await expect(page.getByRole("alert")).toHaveCount(0);
		}

		// CSV export of the selected tab: the browser receives a file.
		await page.getByRole("tab", { name: label.tabs.appointments }).click();
		const downloadPromise = page.waitForEvent("download");
		await page.getByRole("button", { name: label.csv }).click();
		const download = await downloadPromise;
		expect(download.suggestedFilename()).toMatch(
			/^reports-appointments-\d{4}-\d{2}-\d{2}_\d{4}-\d{2}-\d{2}\.csv$/,
		);
	});

	test("denies a user without READ_REPORTS", async ({ page }) => {
		test.skip(
			!noPermUsername || !noPermPassword,
			"Set E2E_NOPERM_USERNAME and E2E_NOPERM_PASSWORD for the denial test",
		);
		await login(page, noPermUsername ?? "", noPermPassword ?? "");
		await page.goto("/reports");
		await expect(
			page.getByText(label.denied).or(page.getByText(label.planLocked)).first(),
		).toBeVisible();
		await expect(page.getByRole("tab")).toHaveCount(0);
	});
});
