import { Navigate, useParams } from "react-router";
import { BookingFlow } from "@/components/booking/booking-flow";
import { PageSpinner } from "@/components/page-spinner";
import { usePortalSession } from "@/lib/use-portal-session";

/** `/:slug/citas/nueva`: booking needs a session and the clinic taking requests. */
export function BookingPage() {
	const { slug = "" } = useParams();
	const session = usePortalSession(slug);

	if (session.status === "signed-out") {
		return <Navigate to={`/${slug}/entrar`} replace />;
	}
	if (session.status === "loading") return <PageSpinner />;
	const { clinic } = session.me;
	if (!clinic.booking?.enabled || !clinic.sections.appointments) {
		return <Navigate to={`/${slug}`} replace />;
	}
	return <BookingFlow clinic={clinic} />;
}

export default BookingPage;
