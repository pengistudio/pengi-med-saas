import React from "react";
import { Navigate, useParams } from "react-router";
import { type Clinic, getClinic } from "@/api/portal-service";
import { NewPatientFlow } from "@/components/new-patient/new-patient-flow";
import { PageSpinner } from "@/components/page-spinner";
import { UnavailablePage } from "@/pages/message-page";

/** `/:slug/primera-cita`: only when the clinic takes new patients. */
export function NewPatientPage() {
	const { slug = "" } = useParams();
	const [clinic, setClinic] = React.useState<Clinic>();
	const [unavailable, setUnavailable] = React.useState(false);

	React.useEffect(() => {
		let active = true;
		getClinic(slug).then((res) => {
			if (!active) return;
			if (res.success) setClinic(res.data);
			else setUnavailable(true);
		});
		return () => {
			active = false;
		};
	}, [slug]);

	if (unavailable) return <UnavailablePage />;
	if (!clinic) return <PageSpinner />;
	if (!clinic.booking?.new_patients) {
		return <Navigate to={`/${slug}/entrar`} replace />;
	}
	return <NewPatientFlow clinic={clinic} />;
}

export default NewPatientPage;
