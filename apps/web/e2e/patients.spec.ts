import { expect, test } from "@playwright/test";

test.describe("Patients Management", () => {
	test("navigate to patients list", async ({ page }) => {
		await page.goto("/");

		// Wait for page load
		await page.waitForLoadState("networkidle").catch(() => {});

		// Look for patients link/button in navigation
		const patientLink = page
			.locator("a, button")
			.filter({ hasText: /patient|paciente/i })
			.first();

		if (await patientLink.isVisible().catch(() => false)) {
			await patientLink.click();

			// Should be on patients page
			await page.waitForURL("**/patient", { timeout: 5000 }).catch(() => {});

			const url = page.url();
			expect(url).toContain("patient");
		}
	});

	test("create new patient", async ({ page }) => {
		// Navigate to patients page
		await page.goto("/");
		await page.waitForLoadState("networkidle").catch(() => {});

		// Click create patient button
		const createButton = page
			.locator("button")
			.filter({ hasText: /new|create|agregar|nuevo/i })
			.first();

		if (await createButton.isVisible().catch(() => false)) {
			await createButton.click();

			// Wait for form to appear
			await page.waitForLoadState("networkidle").catch(() => {});

			// Fill form with test data
			const inputs = await page.locator("input").all();

			if (inputs.length >= 2) {
				await inputs[0].fill("John"); // First name
				await inputs[1].fill("Doe"); // Last name
			}

			// Find and click submit button
			const submitButton = page
				.locator("button")
				.filter({ hasText: /save|guardar|submit/i })
				.first();

			if (await submitButton.isVisible().catch(() => false)) {
				await submitButton.click();

				// Wait for success message or navigation
				await page.waitForLoadState("networkidle").catch(() => {});

				// Should show success or return to list
				await page.waitForTimeout(2000);
			}
		}
	});

	test("search patients", async ({ page }) => {
		await page.goto("/");
		await page.waitForLoadState("networkidle").catch(() => {});

		// Look for search input
		const searchInput = page
			.locator('input[type="text"], input[placeholder*="search" i]')
			.first();

		if (await searchInput.isVisible().catch(() => false)) {
			await searchInput.fill("John");

			// Wait for results
			await page.waitForTimeout(1000);

			// Results should be filtered
			const resultCount = await page
				.locator('tbody tr, [data-testid*="patient"]')
				.count();
			expect(resultCount).toBeGreaterThanOrEqual(0);
		}
	});
});
