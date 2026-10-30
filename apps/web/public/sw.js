// Caches only the app shell's static assets (hashed files under /assets/).
// Everything else — the API, tenant data, PDFs/XML — goes straight to the
// network and is never stored: clinical data must not outlive the session.
const CACHE = "shell-v1";

self.addEventListener("install", () => self.skipWaiting());

self.addEventListener("activate", (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((keys) =>
				Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))),
			)
			.then(() => self.clients.claim()),
	);
});

self.addEventListener("fetch", (event) => {
	const { request } = event;
	const url = new URL(request.url);
	if (
		request.method !== "GET" ||
		url.origin !== self.location.origin ||
		!url.pathname.startsWith("/assets/")
	) {
		return;
	}
	// Hashed filenames are immutable: cache-first is safe.
	event.respondWith(
		caches.open(CACHE).then(async (cache) => {
			const hit = await cache.match(request);
			if (hit) return hit;
			const res = await fetch(request);
			if (res.ok) cache.put(request, res.clone());
			return res;
		}),
	);
});
