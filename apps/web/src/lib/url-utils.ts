/**
 * Retrieves the base domain from the current window location.
 * For example, if current is "tenant.example.com", it returns "example.com".
 * If it's a localhost, it just returns "localhost".
 *
 * @returns {string} The base domain
 */
export function getBaseDomain(): string {
	const currentUrl = new URL(window.location.href);
	const hostnameParts = currentUrl.hostname.split(".");

	if (currentUrl.hostname === "localhost") {
		return "localhost";
	}

	return hostnameParts.length > 2
		? hostnameParts.slice(1).join(".")
		: currentUrl.hostname;
}

/**
 * Redirects the user to a specific tenant subdomain, transferring the local and session storage
 * over via a URL hash.
 *
 * @param slug - The slug of the tenant to redirect to
 */
export function redirectToSubdomain(slug: string) {
	// Capture all local and session storage
	const sessionItems: Record<string, string> = {};
	for (let i = 0; i < sessionStorage.length; i++) {
		const key = sessionStorage.key(i);
		if (key) sessionItems[key] = sessionStorage.getItem(key) || "";
	}
	const localItems: Record<string, string> = {};
	for (let i = 0; i < localStorage.length; i++) {
		const key = localStorage.key(i);
		if (key) localItems[key] = localStorage.getItem(key) || "";
	}
	const storePayload = btoa(
		encodeURIComponent(
			JSON.stringify({ session: sessionItems, local: localItems }),
		),
	);

	const currentUrl = new URL(window.location.href);

	if (currentUrl.hostname === "localhost") {
		window.location.href = `/?tenant=${slug}#store=${storePayload}`;
	} else {
		const tld = getBaseDomain();
		const newUrl = `${currentUrl.protocol}//${slug}.${tld}${currentUrl.port ? `:${currentUrl.port}` : ""}/#store=${storePayload}`;
		window.location.href = newUrl;
	}
}

/**
 * Redirects the user to the root domain, typically used for logging out and
 * clearing tenant-specific contexts.
 */
export function redirectToRootDomain() {
	const currentUrl = new URL(window.location.href);

	if (currentUrl.hostname === "localhost") {
		// Just clear the tenant query param if we're on localhost
		window.location.href = "/login";
	} else {
		const tld = getBaseDomain();
		// If we are already on the root domain, just navigate to login natively
		// to avoid full page reloads if not necessary, but since we are modifying window.location,
		// it will reload anyway which ensures state is cleanly wiped.
		const newUrl = `${currentUrl.protocol}//${tld}${currentUrl.port ? `:${currentUrl.port}` : ""}/login`;
		window.location.href = newUrl;
	}
}
