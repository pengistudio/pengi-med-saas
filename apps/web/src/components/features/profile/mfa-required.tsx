import { Text } from "@pengi/ui";
import { ShieldAlert } from "lucide-react";
import React from "react";
import { useLocation, useNavigate } from "react-router";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { MFA_ENROL_PATH, MFA_ENROL_URL } from "@/lib/mfa-required";
import { selectMfaRequired, useSessionStore } from "@/store/session-store";

/**
 * Sends the user to the profile's security section while their clinic
 * requires two-step verification and they have none (a tenant request
 * answered E-AUTH-026). Never navigates from the profile itself, so there is
 * no loop; enrolling clears the flag (two-factor and passkey services).
 *
 * The refused requests were held, never settled (see api/index.ts), so
 * layout loaders that dedupe in-flight calls or chain their next poll stay
 * stuck. After enrolling, the first navigation off the profile is a full
 * page load, which starts them again. Not on enrolment itself: the
 * two-factor card is showing the one-time recovery codes then.
 */
export function MFARequiredGuard() {
	const mfaRequired = useSessionStore(selectMfaRequired);
	const { pathname, search } = useLocation();
	const navigate = useNavigate();
	const held = React.useRef(false);

	React.useEffect(() => {
		if (mfaRequired) {
			held.current = true;
			if (pathname !== MFA_ENROL_PATH) {
				navigate(MFA_ENROL_URL, { replace: true });
			}
			return;
		}
		if (held.current && pathname !== MFA_ENROL_PATH) {
			held.current = false;
			window.location.assign(pathname + search);
		}
	}, [mfaRequired, pathname, search, navigate]);

	return null;
}

/** The one notice on the profile while enrolment is required. */
export function MFARequiredBanner() {
	const mfaRequired = useSessionStore(selectMfaRequired);
	if (!mfaRequired) return null;
	return (
		<Alert variant="destructive">
			<ShieldAlert />
			<AlertTitle>
				<Text uuid="profile.mfa_required.title" />
			</AlertTitle>
			<AlertDescription>
				<Text uuid="profile.mfa_required.description" />
			</AlertDescription>
		</Alert>
	);
}
