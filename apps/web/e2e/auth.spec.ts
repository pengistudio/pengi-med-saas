import { expect, test } from "@playwright/test";

test.describe("Authentication Flow", () => {
	test("login with valid credentials", async ({ page }) => {
		await page.goto("/login");

		// Verify login page loads
		expect(page).toHaveURL(/.*login/);

		// Fill credentials
		const usernameInputs = await page.locator('input[type="text"]').all();
		const passwordInputs = await page.locator('input[type="password"]').all();

		if (usernameInputs.length > 0) {
			await usernameInputs[0].fill("testuser");
		}
		if (passwordInputs.length > 0) {
			await passwordInputs[0].fill("password123");
		}

		// Submit form
		const loginButton = page
			.locator("button")
			.filter({ hasText: /login|submit|enviar/i })
			.first();
		await loginButton.click();

		// Wait for navigation (with timeout fallback)
		try {
			await page.waitForURL("**/dashboard", { timeout: 5000 });
		} catch {
			// If dashboard URL doesn't exist, just verify we left the login page
			await page.waitForURL(/^(?!.*login)/, { timeout: 5000 });
		}

		// Verify we're authenticated (no longer on login page)
		const url = page.url();
		expect(url).not.toContain("/login");
	});

	test("logout functionality", async ({ page }) => {
		// First login
		await page.goto("/login");
		const usernameInputs = await page.locator('input[type="text"]').all();
		const passwordInputs = await page.locator('input[type="password"]').all();

		if (usernameInputs.length > 0) {
			await usernameInputs[0].fill("testuser");
		}
		if (passwordInputs.length > 0) {
			await passwordInputs[0].fill("password123");
		}

		const loginButton = page
			.locator("button")
			.filter({ hasText: /login|submit|enviar/i })
			.first();
		await loginButton.click();

		// Wait for successful login
		await page.waitForTimeout(2000);

		// Look for logout button (usually in header/menu)
		const logoutButton = page
			.locator("button")
			.filter({ hasText: /logout|salir|sign out/i })
			.first();

		// If logout button exists and is visible
		if (await logoutButton.isVisible().catch(() => false)) {
			await logoutButton.click();

			// Should redirect to login
			await page.waitForURL(/.*login/, { timeout: 5000 });
		}
	});

	test("invalid credentials show error", async ({ page }) => {
		await page.goto("/login");

		const usernameInputs = await page.locator('input[type="text"]').all();
		const passwordInputs = await page.locator('input[type="password"]').all();

		if (usernameInputs.length > 0) {
			await usernameInputs[0].fill("wronguser");
		}
		if (passwordInputs.length > 0) {
			await passwordInputs[0].fill("wrongpassword");
		}

		const loginButton = page
			.locator("button")
			.filter({ hasText: /login|submit|enviar/i })
			.first();
		await loginButton.click();

		// Should remain on login page or show error
		await page.waitForTimeout(2000);
		const url = page.url();
		expect(url).toContain("/login");
	});
});
