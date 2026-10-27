import { useText } from "@pengi/shared";
import { Spinner } from "@pengi/ui";
import React from "react";
import { Link, Navigate, useNavigate, useParams } from "react-router";
import { type Clinic, getClinic } from "@/api/portal-service";
import { ClinicBrand } from "@/components/clinic-brand";
import { SignInForm } from "@/components/sign-in-form";
import { UnavailablePage } from "@/pages/message-page";
import { useSessionStore } from "@/store/session-store";

export function SignInPage() {
	const { slug = "" } = useParams();
	const { textGet } = useText();
	const navigate = useNavigate();
	const setAccessToken = useSessionStore((s) => s.setAccessToken);
	const accessToken = useSessionStore((s) => s.accessToken);
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

	if (accessToken) return <Navigate to={`/${slug}`} replace />;
	if (unavailable) return <UnavailablePage />;
	if (!clinic) {
		return (
			<div className="flex min-h-screen items-center justify-center">
				<Spinner />
			</div>
		);
	}

	return (
		// Top-aligned on a phone, so the keyboard never covers the field.
		<div className="flex min-h-screen justify-center px-4 pt-16 sm:items-center sm:pt-0">
			<main className="w-full max-w-sm">
				<ClinicBrand
					slug={clinic.slug}
					name={clinic.name}
					hasLogo={clinic.has_logo}
				/>
				<h1 className="pt-10 pb-6 text-2xl font-semibold tracking-tight">
					{textGet("portal.signin.title")}
				</h1>
				<SignInForm
					slug={slug}
					onSignedIn={(token) => {
						setAccessToken(token);
						navigate(`/${slug}`, { replace: true });
					}}
				/>
				{clinic.booking?.new_patients && (
					<p className="mt-8 border-t pt-6 text-center text-sm text-muted-foreground">
						{textGet("portal.new_patient.signin_prompt", {
							clinic: clinic.name,
						})}{" "}
						<Link
							to={`/${slug}/primera-cita`}
							className="rounded-sm font-medium text-primary underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
						>
							{textGet("portal.new_patient.signin_link")}
						</Link>
					</p>
				)}
			</main>
		</div>
	);
}
