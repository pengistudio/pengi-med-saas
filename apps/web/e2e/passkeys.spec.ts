import { expect, type Page, test } from "@playwright/test";

/**
 * Passkeys, end to end against a running stack, with Chromium's CDP virtual
 * authenticator standing in for the device: register from the profile, sign
 * in with it, and the post-login offer.
 *
 * Needs the same user the other specs use (it must have NO passkeys and no
 * 2FA; every test removes what it creates):
 *   E2E_USERNAME, E2E_PASSWORD
 *   E2E_ENVIRONMENT (optional)   clinic name to pick when the user has several
 */
const username = process.env.E2E_USERNAME;
const password = process.env.E2E_PASSWORD;
const environment = process.env.E2E_ENVIRONMENT;

// UI labels in either interface language (es / en).
const label = {
	username: /Usuario o correo|Username or email/,
	currentPassword: /Contraseña actual|Current password/,
	twoFactorTitle: /Verificación en dos pasos|Two-step verification/,
	passkeysTitle: /^Passkeys$/,
	addPasskey: /^(Agregar passkey|Add passkey)$/,
	passkeyName: /Nombre de la passkey|Passkey name/,
	continue: /^(Continuar|Continue)$/,
	deletePasskey: /^(Eliminar|Delete) passkey /,
	deleteConfirm: /^(Eliminar|Delete)$/,
	empty: /Aún no tienes passkeys registradas|You have no passkeys yet/,
	loginWithPasskey: /Iniciar sesión con una passkey|Sign in with a passkey/,
	offerTitle:
		/Inicia sesión más rápido con una passkey|Sign in faster with a passkey/,
	offerCreate: /^(Crear passkey|Create passkey)$/,
	offerDontAsk: /^(No volver a preguntar|Don't ask again)$/,
};

// Keys of apps/web/src/lib/passkey-offer-storage.ts.
const OFFER_PENDING_KEY = "passkey-offer:pending";
const OFFER_PREFIX = "passkey-offer:";

// ── Virtual authenticator ──

/** A platform authenticator that always "touches" and verifies the user. */
async function addVirtualAuthenticator(page: Page) {
	const client = await page.context().newCDPSession(page);
	await client.send("WebAuthn.enable");
	await client.send("WebAuthn.addVirtualAuthenticator", {
		options: {
			protocol: "ctap2",
			transport: "internal",
			hasResidentKey: true,
			hasUserVerification: true,
			isUserVerified: true,
			automaticPresenceSimulation: true,
		},
	});
}

/**
 * Keeps the app from leaving the "offer a passkey" mark at a password login,
 * so the dialog does not cover the page in the tests that are not about it.
 */
async function suppressOffer(page: Page) {
	await page.addInitScript((key) => {
		const setItem = Storage.prototype.setItem;
		Storage.prototype.setItem = function (k: string, v: string) {
			if (k === key) return;
			setItem.call(this, k, v);
		};
	}, OFFER_PENDING_KEY);
}

// ── Session helpers ──

async function submitPassword(page: Page) {
	await page.goto("/login");
	await page.getByLabel(label.username).fill(username ?? "");
	await page
		.locator('input[type="password"]')
		.first()
		.fill(password ?? "");
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
			.then(() => choice.click({ timeout: 3000 }))
			.catch(() => {});
		await page.waitForURL((url) => !url.pathname.startsWith("/login"));
	}
}

/**
 * Signs out locally: token, clinic and refresh cookie gone. The offer's
 * per-device choices (dismissed / snoozed) survive, as in the real sign-out.
 */
async function signOut(page: Page) {
	await page.evaluate((prefix) => {
		const kept: [string, string][] = [];
		for (let i = 0; i < localStorage.length; i++) {
			const key = localStorage.key(i);
			if (key?.startsWith(prefix)) {
				kept.push([key, localStorage.getItem(key) ?? ""]);
			}
		}
		localStorage.clear();
		sessionStorage.clear();
		for (const [key, value] of kept) localStorage.setItem(key, value);
	}, OFFER_PREFIX);
	await page.context().clearCookies();
}

/** Forgets the offer's per-device choices (leaves the browser as found). */
async function clearOfferChoices(page: Page) {
	await page.evaluate((prefix) => {
		for (const key of Object.keys(localStorage)) {
			if (key.startsWith(prefix)) localStorage.removeItem(key);
		}
	}, OFFER_PREFIX);
}

async function openSecurity(page: Page) {
	await page.goto("/profile?section=security");
	await expect(page.getByText(label.passkeysTitle).first()).toBeVisible();
}

/** Opens the security section, signing in with the password if needed. */
async function openSecuritySignedIn(page: Page) {
	await page.goto("/profile?section=security");
	if (page.url().includes("/login")) {
		await submitPassword(page);
		await waitSignedIn(page);
		await page.goto("/profile?section=security");
	}
	await expect(page.getByText(label.passkeysTitle).first()).toBeVisible();
}

/** Removes every passkey of the user from the profile. */
async function removeAllPasskeys(page: Page) {
	await openSecuritySignedIn(page);
	const deleteButtons = page.getByRole("button", { name: label.deletePasskey });
	// The list loads after the card renders: wait for it or the empty state.
	await expect(
		deleteButtons.first().or(page.getByText(label.empty)),
	).toBeVisible();
	while ((await deleteButtons.count()) > 0) {
		const before = await deleteButtons.count();
		await deleteButtons.first().click();
		await page
			.getByRole("alertdialog")
			.getByRole("button", { name: label.deleteConfirm })
			.click();
		await expect(deleteButtons).toHaveCount(before - 1);
	}
	await expect(page.getByText(label.empty)).toBeVisible();
}

/** Adds a passkey from the profile (current password dialog). */
async function addPasskeyFromProfile(page: Page, name: string) {
	await page.getByRole("button", { name: label.addPasskey }).click();
	const dialog = page.getByRole("dialog");
	await dialog.getByLabel(label.passkeyName).fill(name);
	await dialog.getByLabel(label.currentPassword).fill(password ?? "");
	await dialog.getByRole("button", { name: label.continue }).click();
	await expect(dialog).toBeHidden();
	await expect(page.getByText(name, { exact: true })).toBeVisible();
}

test.describe("Passkeys", () => {
	test.skip(
		!username || !password,
		"Set E2E_USERNAME and E2E_PASSWORD to run against a real stack",
	);
	test.skip(
		({ browserName }) => browserName !== "chromium",
		"The CDP virtual authenticator only exists in Chromium",
	);
	// One user shared by every test: they must not run in parallel.
	test.describe.configure({ mode: "serial" });
	test.setTimeout(120_000);

	test.beforeEach(async ({ page }) => {
		await addVirtualAuthenticator(page);
	});

	test("register a passkey from the profile and remove it", async ({
		page,
	}) => {
		await suppressOffer(page);
		try {
			await submitPassword(page);
			await waitSignedIn(page);
			await openSecurity(page);
			await removeAllPasskeys(page);

			await addPasskeyFromProfile(page, "e2e-profile-passkey");

			// It persists: listed again after a reload.
			await page.reload();
			await expect(
				page.getByText("e2e-profile-passkey", { exact: true }),
			).toBeVisible();
		} finally {
			await removeAllPasskeys(page);
		}
	});

	test("sign in with the passkey (no second-step prompt)", async ({ page }) => {
		await suppressOffer(page);
		try {
			await submitPassword(page);
			await waitSignedIn(page);
			await openSecurity(page);
			await removeAllPasskeys(page);
			await addPasskeyFromProfile(page, "e2e-login-passkey");

			await signOut(page);
			await page.goto("/login");
			await page.getByRole("button", { name: label.loginWithPasskey }).click();
			await waitSignedIn(page);

			// Smoke check only: the e2e user has no 2FA, so this cannot prove a
			// passkey skips the TOTP step; it guards against a stray prompt.
			await expect(page.getByText(label.twoFactorTitle)).toHaveCount(0);
			await expect(page).not.toHaveURL(/\/login/);

			// The session works: the profile loads with the passkey, used.
			await openSecurity(page);
			await expect(
				page.getByText("e2e-login-passkey", { exact: true }),
			).toBeVisible();
		} finally {
			await removeAllPasskeys(page);
		}
	});

	test("the post-login offer registers a passkey with no password prompt", async ({
		page,
	}) => {
		try {
			// Clean start: no passkeys, no earlier "don't ask again".
			await suppressOffer(page);
			await submitPassword(page);
			await waitSignedIn(page);
			await removeAllPasskeys(page);
			await clearOfferChoices(page);
			await signOut(page);

			// A fresh page without the suppression script: the real behavior.
			const fresh = await page.context().newPage();
			await addVirtualAuthenticator(fresh);
			await submitPassword(fresh);
			await waitSignedIn(fresh);

			const offer = fresh.getByRole("dialog");
			await expect(offer.getByText(label.offerTitle)).toBeVisible();
			await offer.getByLabel(label.passkeyName).fill("e2e-offer-passkey");
			await offer.getByRole("button", { name: label.offerCreate }).click();

			// Fresh session: no password field, the dialog just closes.
			await expect(offer.getByLabel(label.currentPassword)).toHaveCount(0);
			await expect(offer).toBeHidden();

			await openSecurity(fresh);
			await expect(
				fresh.getByText("e2e-offer-passkey", { exact: true }),
			).toBeVisible();
			await fresh.close();
		} finally {
			await removeAllPasskeys(page);
		}
	});

	test("'Don't ask again' keeps the offer away on the next login", async ({
		page,
	}) => {
		try {
			await suppressOffer(page);
			await submitPassword(page);
			await waitSignedIn(page);
			await removeAllPasskeys(page);
			await clearOfferChoices(page);
			await signOut(page);

			const fresh = await page.context().newPage();
			await addVirtualAuthenticator(fresh);
			await submitPassword(fresh);
			await waitSignedIn(fresh);
			const offer = fresh.getByRole("dialog");
			await expect(offer.getByText(label.offerTitle)).toBeVisible();
			await offer.getByRole("button", { name: label.offerDontAsk }).click();
			await expect(offer).toBeHidden();

			// Next password login on this device: no offer.
			await signOut(fresh);
			await submitPassword(fresh);
			await waitSignedIn(fresh);
			// Positive signal first: the app consumed the offer's pending mark, so
			// it did look at the offer and decided not to show it.
			await expect(fresh.getByRole("navigation").first()).toBeVisible();
			await expect
				.poll(() =>
					fresh.evaluate(
						(key) => sessionStorage.getItem(key),
						OFFER_PENDING_KEY,
					),
				)
				.toBeNull();
			await expect(fresh.getByText(label.offerTitle)).toHaveCount(0);
			await clearOfferChoices(fresh);
			await fresh.close();
		} finally {
			await clearOfferChoices(page);
			await removeAllPasskeys(page);
		}
	});

	// The "your clinic requires two-step verification" banner/guard needs a user
	// of the same clinic WITHOUT 2FA or passkey while the policy is on, and
	// turning the policy on needs the acting user to have one themselves
	// (tenant.security.require_mfa.own_mfa_missing). The local DB has a single
	// e2e user, so there is no second, unprotected user to be sent to the
	// profile. Faking it (e.g. by removing the passkey with the policy on) would
	// leave the clinic locked.
	test.skip("MFA required: a user without 2FA or passkey is sent to the profile banner", async () => {});
});
