import { AxiosError, type InternalAxiosRequestConfig } from "axios";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useSessionStore } from "@/store/session-store";
import { guestConfig, portalApi, refreshAccessToken } from "./http-clients";

type Answer = { status: number; data?: unknown };

/** Routes portalApi requests to `answer` and records each request's config. */
function fakeApi(answer: (config: InternalAxiosRequestConfig) => Answer) {
	const calls: InternalAxiosRequestConfig[] = [];
	portalApi.defaults.adapter = async (config: InternalAxiosRequestConfig) => {
		calls.push(config);
		const { status, data = {} } = answer(config);
		const response = { data, status, statusText: "", headers: {}, config };
		if (status >= 400) {
			throw new AxiosError("fail", "ERR_BAD_RESPONSE", config, null, response);
		}
		return response;
	};
	return calls;
}

const auth = (config: InternalAxiosRequestConfig) =>
	config.headers.Authorization as string | undefined;

const originalAdapter = portalApi.defaults.adapter;

describe("portalApi", () => {
	beforeEach(() => useSessionStore.getState().clear());
	afterEach(() => {
		portalApi.defaults.adapter = originalAdapter;
		vi.restoreAllMocks();
	});

	it("sends the session's access token and the interface language", async () => {
		useSessionStore.getState().setAccessToken("access-1");
		const calls = fakeApi(() => ({ status: 200 }));
		await portalApi.get("/me");
		expect(auth(calls[0])).toBe("Bearer access-1");
		expect(calls[0].headers["Accept-Language"]).toBeTruthy();
		expect(calls[0].withCredentials).toBe(true);
	});

	it("sends a guest token as is, never the session's", async () => {
		useSessionStore.getState().setAccessToken("access-1");
		const calls = fakeApi(() => ({ status: 200 }));
		await portalApi.get("/booking/options", guestConfig("guest-1"));
		expect(auth(calls[0])).toBe("Bearer guest-1");
	});

	it("refreshes once on a 401 and retries with the new token", async () => {
		useSessionStore.getState().setAccessToken("stale");
		const calls = fakeApi((config) => {
			if (config.url === "/refresh") {
				return { status: 200, data: { data: { access_token: "fresh" } } };
			}
			return auth(config) === "Bearer fresh"
				? { status: 200, data: { ok: true } }
				: { status: 401 };
		});
		const res = await portalApi.get("/documents");
		expect(res.data).toEqual({ ok: true });
		expect(calls.map((c) => c.url)).toEqual([
			"/documents",
			"/refresh",
			"/documents",
		]);
		expect(useSessionStore.getState().accessToken).toBe("fresh");
	});

	it("ends the session when the refresh fails", async () => {
		useSessionStore.getState().setAccessToken("stale");
		const calls = fakeApi(() => ({ status: 401 }));
		await expect(portalApi.get("/invoices")).rejects.toBeInstanceOf(AxiosError);
		expect(calls.map((c) => c.url)).toEqual(["/invoices", "/refresh"]);
		expect(useSessionStore.getState().accessToken).toBeUndefined();
	});

	it("never refreshes on the sign-in endpoints or a guest request", async () => {
		const calls = fakeApi(() => ({ status: 401 }));
		await expect(portalApi.post("/clinics/sol/verify")).rejects.toBeTruthy();
		await expect(portalApi.post("/login")).rejects.toBeTruthy();
		await expect(
			portalApi.post("/new-patient/requests", {}, guestConfig("guest-1")),
		).rejects.toBeTruthy();
		expect(calls.map((c) => c.url)).not.toContain("/refresh");
	});

	it("shares one refresh between parallel callers", async () => {
		const calls = fakeApi(() => ({
			status: 200,
			data: { data: { access_token: "fresh" } },
		}));
		const [a, b] = await Promise.all([
			refreshAccessToken(),
			refreshAccessToken(),
		]);
		expect(a).toBe("fresh");
		expect(b).toBe("fresh");
		expect(calls).toHaveLength(1);
	});
});
