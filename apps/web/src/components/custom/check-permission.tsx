import { useText } from "@pengi/shared";
import { Button, EmptyState } from "@pengi/ui";
import type React from "react";
import { Outlet, useNavigate } from "react-router";
import { PageSkeleton } from "@/components/custom/page-skeleton";
import { isFeatureEnabled } from "@/config/nav-config";
import useAuth from "@/hooks/use-auth";
import { useEnabledFeatures } from "@/hooks/use-enabled-features";
import usePermission from "@/hooks/use-permission";
import { selectPermissionsFresh, useSessionStore } from "@/store/session-store";

type CheckPermissionProps = {
	children?: React.ReactNode;
	permissions: string[];
	/**
	 * Plan feature (key of `EnabledFeatures`) the route belongs to. When the
	 * tenant's plan doesn't include it, the "not in your plan" screen shows
	 * instead of the page, before any role-permission check.
	 */
	feature?: string;
	/** `feature` is an add-on: off unless the plan says `true`. */
	featureOptIn?: boolean;
};

const CheckPermission = (props: CheckPermissionProps) => {
	const { children, permissions, feature, featureOptIn } = props;
	const { checkPermission } = usePermission();
	const { token } = useAuth();
	const hasPermissions = checkPermission(permissions);
	const permissionsFresh = useSessionStore(selectPermissionsFresh);
	// Persisted permissions may predate a deploy: don't deny until the
	// bootstrap refresh has had its say.
	const waitingForRefresh = !hasPermissions && !permissionsFresh && !!token;
	const { textGet } = useText();
	const navigate = useNavigate();
	const enabledFeatures = useEnabledFeatures();
	const lockedByPlan =
		!!feature && !isFeatureEnabled(enabledFeatures, feature, featureOptIn);

	if (waitingForRefresh) return <PageSkeleton />;
	if (lockedByPlan) {
		if (!token) return null;
		return (
			<EmptyState
				className="min-h-[60vh] justify-center"
				illustration="locked-feature"
				title={textGet("error.plan.title")}
				description={textGet("error.plan.description")}
				action={
					<Button onClick={() => navigate("/subscription")}>
						{textGet("error.plan.see_plans")}
					</Button>
				}
			/>
		);
	}
	if (!hasPermissions) {
		// Logged out (no token): the auth guard is already redirecting.
		if (!token) return null;
		return (
			<EmptyState
				className="min-h-[60vh] justify-center"
				illustration="no-access"
				title={textGet("error.permission.title")}
				description={textGet("error.permission.description")}
				action={
					<Button onClick={() => navigate("/")}>
						{textGet("error.permission.back_home")}
					</Button>
				}
			/>
		);
	}

	return (
		<>
			{children}
			<Outlet />
		</>
	);
};

export default CheckPermission;
