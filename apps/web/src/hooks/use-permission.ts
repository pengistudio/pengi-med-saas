import React from "react";
import { selectEnvironment, useSessionStore } from "@/store/session-store";

/**
 * @returns `checkPermission` - Function to check if the user has the required permissions.
 * @returns `permissions` - Array of permissions the user has.
 */
const usePermission = () => {
	const environment = useSessionStore(selectEnvironment);

	/**
	 * @param featurePermissions - Array of permissions to check.
	 * @returns `true` if the user has all the required permissions, `false` otherwise.
	 */
	const checkPermission = React.useCallback(
		(featurePermissions: string[]): boolean => {
			if (!environment?.permissions) return false;
			const permissionList = environment.permissions;
			return featurePermissions.every((permission) =>
				permissionList.includes(permission),
			);
		},
		[environment],
	);

	return {
		/**
		 * Checks if the user has all the specified permissions.
		 * @param featurePermissions - Array of permission strings to verify.
		 * @returns `true` if the user has all the required permissions, `false` otherwise.
		 */
		checkPermission,
		/**
		 * List of permissions the user has in the current environment.
		 */
		permissions: environment?.permissions || [],
	};
};

export default usePermission;
