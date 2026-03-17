import { extendSessionWithToken, userLogin } from "@/api/auth-service";
import type { ResponseError } from "@/api/fetch";
import { resetSessionExpiredFlag } from "@/api/index";
import { useTokenStore } from "@/store/token-store";
import { useUserStore } from "@/store/user-store";
import useToast from "./use-toast";

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
		localStorage.clear();
		sessionStorage.clear();
		useTokenStore.getState().setToken(undefined);
		useUserStore.getState().clean();
		window.location.href = "/login";
	};

	const refreshExtendToken = async (): Promise<AuthResponse> => {
		if (!token) return { token: null, user_id: null };

		const result = await extendSessionWithToken();

		if (result.success) {
			const refresh = result.data;
			useTokenStore.getState().setToken(refresh.token);
			return { token: refresh.token, user_id: refresh.user_id };
		}

		return handleApiError(result.data);
	};

	return {
		token,
		login,
		logout,
		refreshExtendToken,
	};
};

export default useAuth;
