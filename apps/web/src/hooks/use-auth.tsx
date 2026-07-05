import { useToast } from "@pengi/ui";
import { logoutRequest, userLogin } from "@/api/auth-service";
import type { ResponseError } from "@/api/fetch";
import { resetSessionExpiredFlag } from "@/api/index";
import { useTokenStore } from "@/store/token-store";
import { useUserStore } from "@/store/user-store";

type AuthResponse = { token: string | null; user_id: number | null };

const useAuth = () => {
	const { errorToast } = useToast();
	const { token } = useTokenStore();

	const handleApiError = (error: ResponseError | undefined): AuthResponse => {
		if (error) errorToast(error);
		return { token: null, user_id: null };
	};

	const login = async (
		user_name: string,
		password: string,
	): Promise<AuthResponse> => {
		resetSessionExpiredFlag();
		const result = await userLogin({ user_name, password });

		if (result.success) {
			const { token, user_id } = result.data;
			useTokenStore.getState().setToken(token);
			return { token, user_id };
		}

		return handleApiError(result.data);
	};

	const logout = () => {
		// Best-effort server-side revocation, fired in the background —
		// must not delay clearing local state, or other in-flight/refiring
		// requests get a window to run against an already-invalid session.
		logoutRequest().catch(() => {});
		localStorage.clear();
		sessionStorage.clear();
		useTokenStore.getState().setToken(undefined);
		useUserStore.getState().clean();
		window.location.href = "/login";
	};

	return {
		token,
		login,
		logout,
	};
};

export default useAuth;
