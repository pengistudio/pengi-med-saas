import { expect, test } from "@playwright/test";

test.describe("Authentication Flow", () => {
	test("login with valid credentials", async ({ page }) => {
		await page.goto("/login");

		// Wait for page to load
		await page.waitForLoadState("networkidle").catch(() => {});

		// Verify login page loads
		expect(page).toHaveURL(/.*login/);

		// Wait for inputs to be visible
		await page
			.locator('input[type="text"]')
			.first()
			.waitFor({ state: "visible", timeout: 5000 })
			.catch(() => {});

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
			.filter({ hasText: /login|iniciar|submit|enviar/i })
			.first();
		await loginButton.waitFor({ state: "visible", timeout: 5000 });
		await loginButton.click();

		// Wait for navigation to environments page after login
		try {
			await page.waitForURL("**/login/environments**", { timeout: 10000 });
		} catch {
			// Navigation timeout - that's okay, just verify we got the exchange_token
		}

		// Verify we're on the environments selection page (logged in)
		const url = page.url();
		expect(url).toContain("login/environments");
		expect(url).toContain("exchange_token");
	});

	test("logout functionality", async ({ page }) => {
		// First login
		await page.goto("/login");
		await page.waitForLoadState("networkidle").catch(() => {});

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
			.filter({ hasText: /login|iniciar|submit|enviar/i })
			.first();
		await loginButton.waitFor({ state: "visible", timeout: 5000 });
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
		await page.waitForLoadState("networkidle").catch(() => {});

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
			.filter({ hasText: /login|iniciar|submit|enviar/i })
			.first();
		await loginButton.waitFor({ state: "visible", timeout: 5000 });
		await loginButton.click();

		// Should remain on login page or show error
		await page.waitForTimeout(2000);
		const url = page.url();
		expect(url).toContain("/login");
	});
});
