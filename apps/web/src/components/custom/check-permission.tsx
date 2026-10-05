import { useText } from "@pengi/shared";
import { useToast } from "@pengi/ui";
import React from "react";
import { Outlet, useNavigate } from "react-router";
import { PageSkeleton } from "@/components/custom/page-skeleton";
import useAuth from "@/hooks/use-auth";
import usePermission from "@/hooks/use-permission";
import { selectPermissionsFresh, useSessionStore } from "@/store/session-store";

type CheckPermissionProps = {
	children?: React.ReactNode;
	permissions: string[];
};

const CheckPermission = (props: CheckPermissionProps) => {
	const { children, permissions } = props;
	const { checkPermission } = usePermission();
	const { token } = useAuth();
	const hasPermissions = checkPermission(permissions);
	const permissionsFresh = useSessionStore(selectPermissionsFresh);
	// Persisted permissions may predate a deploy: don't deny until the
	// bootstrap refresh has had its say.
	const waitingForRefresh = !hasPermissions && !permissionsFresh && !!token;
	const navigate = useNavigate();
	const { infoToast } = useToast();
	const { textGet } = useText();

	React.useEffect(() => {
		// Only show permission error if user has a token (is logged in)
		// If no token, user is logging out or already logged out
		if (!hasPermissions && token && permissionsFresh) {
			infoToast(
				textGet("error.permission.title") || "No puedes acceder a esta ruta",
				{
					description:
						textGet("error.permission.description") ||
						"No tienes los permisos suficientes",
				},
			);
			navigate("/");
		}
	}, [hasPermissions, permissionsFresh, navigate, infoToast, token, textGet]);

	if (waitingForRefresh) return <PageSkeleton />;
	if (!hasPermissions) return null;

	return (
		<>
			{children}
			<Outlet />
		</>
	);
};

export default CheckPermission;
