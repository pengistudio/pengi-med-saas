import { useText } from "@pengi/shared";
import { Spinner } from "@pengi/ui";
import React from "react";
import { Navigate, useParams } from "react-router";
import { refreshAccessToken } from "@/api/http-clients";
import { getMe, logout, type Me } from "@/api/portal-service";
import { HomeContent } from "@/components/home-content";
import { useSessionStore } from "@/store/session-store";

/** Home: needs a session; restores it from the refresh cookie on a reload. */
export function HomePage() {
	const { slug = "" } = useParams();
	const { textGet } = useText();
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

	const onLogout = async () => {
		await logout();
		clear();
	};

	if (!accessToken && !restoring) {
		return <Navigate to={`/${slug}/entrar`} replace />;
	}
	if (!me) {
		return (
			<div className="flex min-h-screen items-center justify-center">
				<Spinner aria-label={textGet("portal.loading")} />
			</div>
		);
	}
	return <HomeContent me={me} onLogout={onLogout} />;
}
