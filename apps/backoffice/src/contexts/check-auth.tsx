import { type JwtPayload, jwtDecode } from "jwt-decode";
import React from "react";
import { useNavigate } from "react-router";
import { refreshToken } from "@/api/auth-service";
import useAuth from "@/hooks/use-auth";
import useToast from "@/hooks/use-toast";
import { ONE_SECOND } from "@/lib/constants";
import { useTokenStore } from "@/store/token-store";

type Props = {
	children?: React.ReactNode;
};

const REFRESH_BUFFER_MS = 60000; // Refresh 60s before expiry

const CheckAuth = (props: Props) => {
	const { children } = props;
	const { token, logout } = useAuth();
	const { setToken } = useTokenStore();
	const { infoToast } = useToast();
	const navigate = useNavigate();

	const decoded = React.useMemo(() => {
		return token ? jwtDecode<JwtPayload>(token) : undefined;
	}, [token]);

	const handleSessionExpired = React.useCallback(async () => {
		if (!token) return;
		const result = await refreshToken();
		if (result.success) {
			setToken(result.data.token);
			return;
		}

		infoToast("Sesión expirada", {
			description: "Tu sesión ha expirado, por favor inicia sesión nuevamente.",
		});

		logout();
		navigate("/login");
	}, [logout, navigate, infoToast, setToken, token]);

	// Schedule a silent refresh before the token expires
	React.useEffect(() => {
		if (!decoded || !decoded.exp) return;

		const now = Date.now();
		const expirationTime = decoded.exp * ONE_SECOND;
		const timeUntilRefresh = expirationTime - now - REFRESH_BUFFER_MS;

		let timeout: ReturnType<typeof setTimeout> | undefined;
		if (timeUntilRefresh > 0) {
			timeout = setTimeout(() => {
				handleSessionExpired();
			}, timeUntilRefresh);
		} else {
			handleSessionExpired();
		}

		return () => {
			if (timeout !== undefined) clearTimeout(timeout);
		};
	}, [decoded, handleSessionExpired]);

	// Redirect to login if no token
	React.useEffect(() => {
		if (!token) {
			navigate("/login");
		}
	}, [navigate, token]);

	return <>{children}</>;
};

export default CheckAuth;
