import { Navigate, useParams } from "react-router";
import { HomeContent } from "@/components/home-content";
import { PageSpinner } from "@/components/page-spinner";
import { usePortalSession } from "@/lib/use-portal-session";

/** Home: needs a session; restores it from the refresh cookie on a reload. */
export function HomePage() {
	const { slug = "" } = useParams();
	const session = usePortalSession(slug);

	if (session.status === "signed-out") {
		return <Navigate to={`/${slug}/entrar`} replace />;
	}
	if (session.status === "loading") return <PageSpinner />;
	return <HomeContent me={session.me} onLogout={session.logout} />;
}
