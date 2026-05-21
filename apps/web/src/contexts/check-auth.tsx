import { type JwtPayload, jwtDecode } from "jwt-decode";
import React from "react";
import { useLocation, useNavigate } from "react-router";
import { refreshToken } from "@/api/auth-service";
import { getMySubscription } from "@/api/subscription-service";
import useAuth from "@/hooks/use-auth";
import { useText } from "@/hooks/use-text";
import useToast from "@/hooks/use-toast";
import { ONE_SECOND } from "@/lib/constants";
import {
	selectSetSubscriptionGraceDaysLeft,
	useSessionStore,
} from "@/store/session-store";
import { useTokenStore } from "@/store/token-store";

type Props = {
	children?: React.ReactNode;
};

const REFRESH_BUFFER_MS = 60000; // Refresh 60s before expiry

const CheckAuth = (props: Props) => {
	const { children } = props;
	const { token, logout } = useAuth();
	const { textGet } = useText();
	const { setToken } = useTokenStore();
	const { infoToast } = useToast();
	const navigate = useNavigate();
	const { pathname } = useLocation();
	const setGraceDaysLeft = useSessionStore(selectSetSubscriptionGraceDaysLeft);

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

		infoToast(textGet("session.expired.title"), {
			description: textGet("session.expired"),
		});

		logout();
		navigate("/login");
	}, [logout, navigate, infoToast, setToken, token, textGet]);

	// Schedule a silent refresh before the token expires
	React.useEffect(() => {
		if (!decoded?.exp) return;

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

	const GRACE_PERIOD_DAYS = 3;

	// Check subscription grace period on load
	React.useEffect(() => {
		if (!token) return;
		getMySubscription().then((res) => {
			if (!res.success) return;
			const { days_left } = res.data;
			if (days_left < -GRACE_PERIOD_DAYS && pathname !== "/subscription") {
				navigate("/subscription");
				return;
			}
			// Only store grace days when within the grace window (-3 to 0)
			setGraceDaysLeft(
				days_left < 0 && days_left >= -GRACE_PERIOD_DAYS ? days_left : 0,
			);
		});
	}, [token, navigate, pathname, setGraceDaysLeft]);

	return <>{children}</>;
};

export default CheckAuth;
