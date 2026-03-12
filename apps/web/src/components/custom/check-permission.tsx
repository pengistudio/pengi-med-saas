import React from "react";
import { Outlet, useNavigate } from "react-router";
import useAuth from "@/hooks/use-auth";
import usePermission from "@/hooks/use-permission";
import { useText } from "@/hooks/use-text";
import useToast from "@/hooks/use-toast";

type CheckPermissionProps = {
	children?: React.ReactNode;
	permissions: string[];
};

const CheckPermission = (props: CheckPermissionProps) => {
	const { children, permissions } = props;
	const { checkPermission } = usePermission();
	const { token } = useAuth();
	const hasPermissions = checkPermission(permissions);
	const navigate = useNavigate();
	const { infoToast } = useToast();
	const { textGet } = useText();

	React.useEffect(() => {
		// Only show permission error if user has a token (is logged in)
		// If no token, user is logging out or already logged out
		if (!hasPermissions && token) {
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
	}, [hasPermissions, navigate, infoToast, token, textGet]);

	if (!hasPermissions) return null;

	return (
		<>
			{children}
			<Outlet />
		</>
	);
};

export default CheckPermission;
