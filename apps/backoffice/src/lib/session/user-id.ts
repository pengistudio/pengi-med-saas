import { jwtDecode } from "jwt-decode";

/** The admin an access token was issued to (claim `userId`), if readable. */
export function userIdOf(token: string | undefined): number | undefined {
	if (!token) return undefined;
	try {
		const { userId } = jwtDecode<{ userId?: unknown }>(token);
		return typeof userId === "number" ? userId : undefined;
	} catch {
		return undefined;
	}
}
