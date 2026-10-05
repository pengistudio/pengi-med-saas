import { getMyEnvironments } from "@/api/user-service";
import { useSessionStore } from "@/store/session-store";
import { useTokenStore } from "@/store/token-store";
import { useUserStore } from "@/store/user-store";

const THROTTLE_MS = 30_000;
let inFlight: Promise<void> | null = null;
let lastForced = 0;

function dropSession() {
	localStorage.clear();
	sessionStorage.clear();
	useSessionStore.getState().clean();
	useTokenStore.getState().setToken(undefined);
	useUserStore.getState().clean();
	window.location.href = "/login";
}

/**
 * Re-reads the stored environment (role permissions, enabled features) from the
 * API, silently. If the environment no longer exists the session is cleared.
 * Any other failure keeps the persisted data and just ends the "stale" wait.
 */
export function refreshEnvironment(): Promise<void> {
	const session = useSessionStore.getState();
	const current = session.environment;
	if (!current || !useTokenStore.getState().token) {
		session.setPermissionsFresh(true);
		return Promise.resolve();
	}
	if (inFlight) return inFlight;

	inFlight = getMyEnvironments()
		.then((res) => {
			const state = useSessionStore.getState();
			if (!res.success) {
				state.setPermissionsFresh(true);
				return;
			}
			const fresh = res.data.find((e) => e.ID === current.id);
			if (!fresh) {
				dropSession();
			} else if (fresh.company?.tenant) {
				state.setEnvironment(fresh);
			} else {
				// The handler omits the company when its lookup fails: transient.
				state.setPermissionsFresh(true);
			}
		})
		.catch(() => useSessionStore.getState().setPermissionsFresh(true))
		.finally(() => {
			inFlight = null;
		});
	return inFlight;
}

/** For 403 permission/plan errors: at most one refresh every 30 s. */
export function refreshEnvironmentThrottled(): void {
	const now = Date.now();
	if (now - lastForced < THROTTLE_MS) return;
	lastForced = now;
	refreshEnvironment();
}
