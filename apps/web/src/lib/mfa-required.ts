/**
 * The clinic requires two-step verification and the user has none: every
 * tenant request answers 403 with this code (TenantMiddleware). The HTTP
 * layer raises `mfaRequired` in the session store and <MFARequiredGuard />
 * sends the user to where they can enrol.
 */
export const MFA_REQUIRED_ERROR = "E-AUTH-026";

/** Where the user enrols (TwoFactorCard, PasskeysCard). */
export const MFA_ENROL_PATH = "/profile";
export const MFA_ENROL_URL = `${MFA_ENROL_PATH}?section=security`;
