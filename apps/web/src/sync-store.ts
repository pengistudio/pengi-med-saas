// This file must be imported at the VERY TOP of main.tsx
// It ensures that any cross-subdomain store data passed via the URL hash
// is restored to localStorage and sessionStorage before Zustand initializes.

try {
	if (window.location.hash.startsWith("#store=")) {
		const encoded = window.location.hash.replace("#store=", "");
		const decoded = JSON.parse(decodeURIComponent(atob(encoded)));

		if (decoded.session) {
			Object.entries(decoded.session).forEach(([k, v]) => {
				sessionStorage.setItem(k, v as string);
			});
		}

		if (decoded.local) {
			Object.entries(decoded.local).forEach(([k, v]) => {
				localStorage.setItem(k, v as string);
			});
		}

		// Remove the #store hash from the URL to clean it up without reloading the page
		window.history.replaceState(
			null,
			"",
			window.location.pathname + window.location.search,
		);
	}
} catch (e) {
	console.error("Failed to sync store cross-subdomain", e);
}
