import React from "react";
import { refreshAccessToken } from "@/api/http-clients";
import { getMe, logout, type Me } from "@/api/portal-service";
import { useSessionStore } from "@/store/session-store";

export type PortalSession =
	| { status: "loading" }
	| { status: "signed-out" }
	| { status: "ready"; me: Me; logout: () => Promise<void> };

/**
 * The signed-in patient of the clinic at `slug`. Restores the session from
 * the refresh cookie on a reload; every page behind sign-in uses it.
 */
export function usePortalSession(slug: string): PortalSession {
	const accessToken = useSessionStore((s) => s.accessToken);
	const clear = useSessionStore((s) => s.clear);
	const [me, setMe] = React.useState<Me>();
	const [restoring, setRestoring] = React.useState(!accessToken);

	React.useEffect(() => {
		if (useSessionStore.getState().accessToken) return;
		refreshAccessToken().finally(() => setRestoring(false));
	}, []);

	React.useEffect(() => {
		if (!accessToken) return;
		let active = true;
		getMe().then((res) => {
			if (!active) return;
			// The refresh cookie is shared by every clinic's URL: a session from
			// another clinic must not open this one. Sign out and ask again.
			if (res.success && res.data.clinic.slug === slug) setMe(res.data);
			else if (res.success) logout().finally(clear);
			else clear();
		});
		return () => {
			active = false;
		};
	}, [accessToken, clear, slug]);

	const signOut = React.useCallback(async () => {
		await logout();
		clear();
	}, [clear]);

	if (!accessToken && !restoring) return { status: "signed-out" };
	if (!me) return { status: "loading" };
	return { status: "ready", me, logout: signOut };
}
