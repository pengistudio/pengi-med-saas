import { type JwtPayload, jwtDecode } from "jwt-decode";
import React from "react";
import { useNavigate } from "react-router";
import { refreshToken } from "@/api/auth-service";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import useAuth from "@/hooks/use-auth";
import { useText } from "@/hooks/use-text";
import useToast from "@/hooks/use-toast";
import { ONE_SECOND, ZERO } from "@/lib/constants";
import { useTokenStore } from "@/store/token-store";

type Props = {
	children?: React.ReactNode;
};
const SIXTY_SECONDS = 60000;
const CheckAuth = (props: Props) => {
	const { children } = props;
	const { token, logout, refreshExtendToken } = useAuth();
	const { textGet } = useText();
	const { setToken } = useTokenStore();
	const { infoToast } = useToast();
	const [isDialogOpen, setDialogOpen] = React.useState(false);
	const [seconds, setSeconds] = React.useState(60);
	const decoded = React.useMemo(() => {
		return token ? jwtDecode<JwtPayload>(token) : undefined;
	}, [token]);

	const navigate = useNavigate();

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

	React.useEffect(() => {
		if (!decoded || !decoded.exp) return;

		const now = Date.now();
		const expirationTime = decoded.exp * ONE_SECOND;
		const timeRemaining = expirationTime - now;

		const warningTime = timeRemaining - SIXTY_SECONDS;

		let timeout: ReturnType<typeof setTimeout> | undefined;
		if (warningTime > 0) {
			timeout = setTimeout(() => {
				setDialogOpen(true);
			}, warningTime);
		} else {
			handleSessionExpired();
		}

		return () => {
			if (timeout !== undefined) clearTimeout(timeout);
		};
	}, [decoded, handleSessionExpired]);

	const handleExtendSession = React.useCallback(async () => {
		setDialogOpen(false);
		try {
			await refreshExtendToken();
			infoToast(textGet("session.extended.title"), {
				description: textGet("session.extended"),
			});
		} catch (_error) {
			infoToast("Error", {
				description: textGet("session.extended.error"),
			});
			handleSessionExpired();
		}
	}, [refreshExtendToken, handleSessionExpired, infoToast, textGet]);

	React.useEffect(() => {
		if (!token) {
			navigate("/login");
		}
	}, [navigate, token]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: Decrement used on every render
	React.useEffect(() => {
		if (!isDialogOpen) return;
		const interval = setInterval(() => {
			decrement();
		}, ONE_SECOND);
		return () => {
			clearInterval(interval);
		};
	}, [isDialogOpen]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: Decrement used on every render
	React.useEffect(() => {
		let lastTime = Date.now();

		const tick = () => {
			const now = Date.now();
			const delta = Math.floor((now - lastTime) / 1000);
			if (delta > 0) {
				for (let i = 0; i < delta; i++) {
					decrement();
				}
			}
			lastTime = now;
		};

		const interval = setInterval(() => {
			if (document.hidden && isDialogOpen) tick();
		}, 1000);

		if (seconds <= ZERO) {
			clearInterval(interval);
			handleSessionExpired();
		}

		return () => clearInterval(interval);
	}, [seconds, handleSessionExpired, isDialogOpen]);

	return (
		<>
			{children}
			<Dialog open={isDialogOpen} onOpenChange={setDialogOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Extender Sesión</DialogTitle>
						<DialogDescription>
							Su sesión está a punto de expirar. ¿Desea extenderla?
						</DialogDescription>
					</DialogHeader>
					<div className="flex flex-row items-center justify-center">
						<p>
							<span className="text-3xl">{seconds}</span>
							<span>s</span>
						</p>
					</div>
					<DialogFooter>
						<Button variant="ghost" onClick={() => handleSessionExpired()}>
							Cerrar sesión
						</Button>
						<Button onClick={handleExtendSession}>Extender sesión</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);

	function decrement() {
		setSeconds((prev) => prev - 1);
	}
};

export default CheckAuth;
