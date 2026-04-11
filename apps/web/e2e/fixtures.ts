import { test as base, type Page } from "@playwright/test";

export const test = base.extend<{ authenticatedPage: Page }>({
	authenticatedPage: async ({ page }, use) => {
		// Login before each test
		await page.goto("/login");

		// Fill login form with test credentials
		await page.fill('input[type="text"]', "testuser");
		await page.fill('input[type="password"]', "password123");

		// Click login button
		await page.click('button:has-text("Login")');

		// Wait for navigation to dashboard
		await page.waitForURL("**/dashboard", { timeout: 10000 }).catch(() => {
			// If dashboard doesn't exist, wait for any successful navigation away from login
			return page.waitForURL("**/!(login)", { timeout: 5000 });
		});

		await use(page);
	},
});

export { expect } from "@playwright/test";
