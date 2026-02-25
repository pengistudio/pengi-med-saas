import React from "react";
import { useSessionStore } from "@/store/session-store";

const usePermission = () => {
	const { environment } = useSessionStore();

	const checkPermission = React.useCallback(
		(featurePermissions: string[]): boolean => {
			if (!environment || !environment.permissions) return false;
			const permissionList = environment.permissions;
			return featurePermissions.every((permission) =>
				permissionList.includes(permission),
			);
		},
		[environment],
	);

	return { checkPermission, permissions: environment?.permissions || [] };
};

export default usePermission;
