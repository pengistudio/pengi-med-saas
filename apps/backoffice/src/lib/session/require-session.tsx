import React from "react";
import { Navigate, useLocation } from "react-router";
import { toast } from "sonner";
import { useStore } from "zustand";
import { useText } from "@/hooks/use-text";
import type { Session } from "./session";

/**
 * Renders its children only with an authenticated session. While the session
 * is being restored it shows a loading state; without one it redirects to
 * /login?next=<current path>, telling the user when their session expired.
 */
export function RequireSession({
	session,
	children,
}: {
	session: Session;
	children: React.ReactNode;
}) {
	const { status, expired } = useStore(session.store);
	const { textGet } = useText();
	const location = useLocation();

	React.useEffect(() => {
		if (status === "anonymous" && expired) {
			toast.info(textGet("backoffice.session.expired.title"), {
				id: "session-expired",
				description: textGet("backoffice.session.expired.description"),
			});
		}
	}, [status, expired, textGet]);

	if (status === "restoring") {
		return (
			<div className="flex h-screen items-center justify-center">
				<p className="text-muted-foreground animate-pulse">
					{textGet("backoffice.common.loading")}
				</p>
			</div>
		);
	}
	if (status === "anonymous") {
		const next = encodeURIComponent(location.pathname + location.search);
		return <Navigate to={`/login?next=${next}`} replace />;
	}
	return <>{children}</>;
}
