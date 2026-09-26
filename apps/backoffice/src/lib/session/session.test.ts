import axios, { AxiosError, type InternalAxiosRequestConfig } from "axios";
import { describe, expect, it } from "vitest";
import type { AuthGateway } from "./session";
import { createSession } from "./session";

/** Fake auth API: refresh succeeds while `cookie` is valid. */
function fakeGateway(opts: { cookie: boolean }) {
	const calls = { refresh: 0, login: 0, logout: 0 };
	let issued = 0;
	const gateway: AuthGateway = {
		login: async (creds) => {
			calls.login++;
			if (creds.password !== "right") return null;
			opts.cookie = true;
			return `access-${++issued}`;
		},
		refresh: async () => {
			calls.refresh++;
			await new Promise((r) => setTimeout(r, 5));
			return opts.cookie ? `access-${++issued}` : null;
		},
		logout: async () => {
			calls.logout++;
			opts.cookie = false;
		},
	};
	return { gateway, calls };
}

/**
 * An axios client whose "server" accepts only `validToken` and records the
 * Authorization header of every request it receives.
 */
function fakeApi(valid: { token: string }) {
	const seen: (string | undefined)[] = [];
	const client = axios.create({
		adapter: async (config: InternalAxiosRequestConfig) => {
			const auth = config.headers.Authorization as string | undefined;
			seen.push(auth);
			const response = {
				data: { ok: true },
				status: 200,
				statusText: "OK",
				headers: {},
				config,
			};
			if (auth !== `Bearer ${valid.token}`) {
				throw new AxiosError("401", "ERR_BAD_REQUEST", config, null, {
					...response,
					status: 401,
					statusText: "Unauthorized",
				});
			}
			return response;
		},
	});
	return { client, seen };
}

describe("session", () => {
	it("restores from the refresh cookie and authorizes requests", async () => {
		const { gateway } = fakeGateway({ cookie: true });
		const session = createSession(gateway);
		const api = fakeApi({ token: "access-1" });
		session.attach(api.client);

		await session.restore();

		expect(session.store.getState().status).toBe("authenticated");
		await expect(api.client.get("/plans")).resolves.toMatchObject({
			status: 200,
		});
		expect(api.seen).toEqual(["Bearer access-1"]);
	});

	it("without a refresh cookie, restores to anonymous (not expired)", async () => {
		const { gateway } = fakeGateway({ cookie: false });
		const session = createSession(gateway);

		await session.restore();

		expect(session.store.getState()).toMatchObject({
			status: "anonymous",
			expired: false,
		});
	});

	it("refreshes once for concurrent 401s and retries every request", async () => {
		const { gateway, calls } = fakeGateway({ cookie: true });
		const session = createSession(gateway);
		const valid = { token: "access-1" };
		const api = fakeApi(valid);
		session.attach(api.client);
		await session.restore();
		valid.token = "access-2"; // the server now rejects access-1

		const results = await Promise.all(
			Array.from({ length: 5 }, () => api.client.get("/plans")),
		);

		expect(results.every((r) => r.status === 200)).toBe(true);
		expect(calls.refresh).toBe(2); // one on restore, one for the five 401s
		expect(session.store.getState().status).toBe("authenticated");
	});

	it("when the refresh fails, the session expires and requests are rejected", async () => {
		const opts = { cookie: true };
		const { gateway } = fakeGateway(opts);
		const session = createSession(gateway);
		const valid = { token: "access-1" };
		const api = fakeApi(valid);
		session.attach(api.client);
		await session.restore();
		valid.token = "nobody-has-this";
		opts.cookie = false;

		const results = await Promise.allSettled([
			api.client.get("/plans"),
			api.client.get("/roles"),
		]);

		expect(results.every((r) => r.status === "rejected")).toBe(true);
		expect(session.store.getState()).toMatchObject({
			status: "anonymous",
			expired: true,
			token: undefined,
		});
	});

	it("logs in and out", async () => {
		const { gateway, calls } = fakeGateway({ cookie: false });
		const session = createSession(gateway);
		await session.restore();

		expect(await session.login({ user_name: "admin", password: "wrong" })).toBe(
			false,
		);
		expect(session.store.getState().status).toBe("anonymous");

		expect(await session.login({ user_name: "admin", password: "right" })).toBe(
			true,
		);
		expect(session.store.getState().status).toBe("authenticated");

		await session.logout();
		expect(calls.logout).toBe(1);
		expect(session.store.getState()).toMatchObject({
			status: "anonymous",
			expired: false,
			token: undefined,
		});
	});
});
