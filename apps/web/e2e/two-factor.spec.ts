import { createHmac } from "node:crypto";
import { expect, type Page, test } from "@playwright/test";

/**
 * Two-factor authentication, end to end against a running stack: turn it on
 * from the profile, sign in with a TOTP code, sign in with a recovery code,
 * turn it off.
 *
 * Needs a dedicated user WITHOUT 2FA, through the environment:
 *   E2E_2FA_USERNAME, E2E_2FA_PASSWORD  a user of the clinic (not the one the
 *                                       other specs use: a failed run can
 *                                       leave it with 2FA on; reset it from
 *                                       the backoffice, company users)
 *   E2E_ENVIRONMENT (optional)          clinic name to pick when the user has
 *                                       several
 */
const username = process.env.E2E_2FA_USERNAME;
const password = process.env.E2E_2FA_PASSWORD;
const environment = process.env.E2E_ENVIRONMENT;

// UI labels in either interface language (es / en).
const label = {
	username: /Usuario o correo|Username or email/,
	password: /^(Contraseña|Password)$/,
	currentPassword: /Contraseña actual|Current password/,
	enable: /^(Activar|Turn on)$/,
	disable: /^(Desactivar|Turn off)$/,
	continue: /^(Continuar|Continue)$/,
	appCode: /Código de la app|Code from the app/,
	verifyAndEnable: /Verificar y activar|Verify and turn on/,
	saved: /Guardé mis códigos|I saved my recovery codes/,
	done: /^(Listo|Done)$/,
	twoFactorTitle: /Verificación en dos pasos|Two-step verification/,
	loginCode: /Código de verificación|Verification code/,
	verify: /^(Verificar|Verify)$/,
	useRecovery: /Usar un código de recuperación|Use a recovery code/,
	recoveryCode: /Código de recuperación|Recovery code/,
	statusOn: /^(Activada|On)$/,
};

// ── RFC 6238 TOTP (SHA-1, 30 s, 6 digits), as authenticator apps compute it ──

const STEP_SECONDS = 30;

function base32Decode(input: string): Buffer {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
	const clean = input.replace(/[\s=]/g, "").toUpperCase();
	let bits = 0;
	let value = 0;
	const out: number[] = [];
	for (const char of clean) {
		const index = alphabet.indexOf(char);
		if (index < 0) throw new Error(`invalid base32 character: ${char}`);
		value = (value << 5) | index;
		bits += 5;
		if (bits >= 8) {
			out.push((value >>> (bits - 8)) & 0xff);
			bits -= 8;
		}
	}
	return Buffer.from(out);
}

function totpAt(secret: string, step: number): string {
	const counter = Buffer.alloc(8);
	counter.writeBigUInt64BE(BigInt(step));
	const hmac = createHmac("sha1", base32Decode(secret))
		.update(counter)
		.digest();
	const offset = hmac[hmac.length - 1] & 0x0f;
	const binary = hmac.readUInt32BE(offset) & 0x7fffffff;
	return String(binary % 1_000_000).padStart(6, "0");
}

const currentStep = () => Math.floor(Date.now() / 1000 / STEP_SECONDS);

/**
 * A code from a step after `lastStep`: the API refuses a code at or before
 * the last accepted step (replay), so wait for the next window if needed.
 */
async function freshCode(
	page: Page,
	secret: string,
	lastStep: number,
): Promise<{ code: string; step: number }> {
	while (currentStep() <= lastStep) {
		await page.waitForTimeout(1000);
	}
	const step = currentStep();
	return { code: totpAt(secret, step), step };
}

// ── Session helpers ──

async function submitPassword(page: Page) {
	await page.goto("/login");
	await page.getByLabel(label.username).fill(username ?? "");
	await page.getByLabel(label.password).fill(password ?? "");
	await page.locator("button[type=submit]").click();
}

/** After a completed sign-in: pick the clinic if asked, land in the app. */
async function waitSignedIn(page: Page) {
	await page.waitForURL(/\/login\/environments|\/$|\/dashboard/);
	if (page.url().includes("/login/environments")) {
		const choice = environment
			? page.getByRole("button", { name: new RegExp(environment) })
			: page.locator("button[type=button]").first();
		await choice
			.waitFor({ state: "visible", timeout: 3000 })
			.then(() => choice.click())
			.catch(() => {});
		await page.waitForURL((url) => !url.pathname.startsWith("/login"));
	}
}

/** Signs out locally: token, clinic and refresh cookie gone. */
async function signOut(page: Page) {
	await page.evaluate(() => {
		localStorage.clear();
		sessionStorage.clear();
	});
	await page.context().clearCookies();
}

async function openSecurity(page: Page) {
	await page.goto("/profile?section=security");
	await expect(page.getByText(label.twoFactorTitle).first()).toBeVisible();
}

async function typeCode(page: Page, name: RegExp, code: string) {
	const firstSlot = page.getByRole("textbox", { name }).first();
	await firstSlot.click();
	await firstSlot.pressSequentially(code);
}

test.describe("Two-factor authentication", () => {
	test.skip(
		!username || !password,
		"Set E2E_2FA_USERNAME and E2E_2FA_PASSWORD to run against a real stack",
	);
	// Waits up to one TOTP window between enabling and signing in.
	test.setTimeout(120_000);

	test("enable, sign in with TOTP, sign in with a recovery code, disable", async ({
		page,
	}) => {
		// 1. Plain sign-in and enrollment from the profile.
		await submitPassword(page);
		await waitSignedIn(page);
		await openSecurity(page);

		await page.getByRole("button", { name: label.enable }).click();
		const enrollDialog = page.getByRole("dialog");
		await enrollDialog.getByLabel(label.currentPassword).fill(password ?? "");
		await enrollDialog.getByRole("button", { name: label.continue }).click();

		const secret = await enrollDialog
			.getByTestId("two-factor-secret")
			.getAttribute("data-secret");
		expect(secret).toBeTruthy();
		const enrolled = await freshCode(page, secret ?? "", 0);
		await typeCode(page, label.appCode, enrolled.code);
		await enrollDialog
			.getByRole("button", { name: label.verifyAndEnable })
			.click();

		const codeItems = enrollDialog.getByTestId("recovery-code");
		await expect(codeItems).toHaveCount(10);
		const recoveryCodes = (await codeItems.allTextContents()).map((c) =>
			c.trim(),
		);
		await enrollDialog.getByRole("checkbox", { name: label.saved }).click();
		await enrollDialog.getByRole("button", { name: label.done }).click();
		await expect(page.getByText(label.statusOn)).toBeVisible();

		// The session survived enabling (the client switched to the new token).
		await page.reload();
		await expect(page.getByText(label.statusOn)).toBeVisible();

		// 2. Sign in again: the password alone asks for the code.
		await signOut(page);
		await submitPassword(page);
		await expect(page.getByText(label.twoFactorTitle).first()).toBeVisible();
		const login = await freshCode(page, secret ?? "", enrolled.step);
		await typeCode(page, label.loginCode, login.code);
		await page.getByRole("button", { name: label.verify }).click();
		await waitSignedIn(page);

		// 3. Sign in with a recovery code instead of the app.
		await signOut(page);
		await submitPassword(page);
		await page.getByRole("button", { name: label.useRecovery }).click();
		await page.getByLabel(label.recoveryCode).fill(recoveryCodes[0]);
		await page.getByRole("button", { name: label.verify }).click();
		await waitSignedIn(page);

		// 4. Turn it off with the password and another recovery code (no wait
		// for a new TOTP window).
		await openSecurity(page);
		await page.getByRole("button", { name: label.disable }).click();
		const disableDialog = page.getByRole("dialog");
		await disableDialog.getByLabel(label.currentPassword).fill(password ?? "");
		await disableDialog.getByLabel(label.loginCode).fill(recoveryCodes[1]);
		await disableDialog.getByRole("button", { name: label.disable }).click();
		await expect(
			page.getByRole("button", { name: label.enable }),
		).toBeVisible();
	});
});
