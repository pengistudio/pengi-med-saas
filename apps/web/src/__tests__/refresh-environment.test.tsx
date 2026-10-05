import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { EnvironmentWithCompany } from "@/types/user-type";

const getMyEnvironments = vi.hoisted(() => vi.fn());
vi.mock("@/api/user-service", () => ({ getMyEnvironments }));
const infoToast = vi.hoisted(() => vi.fn());
vi.mock("@pengi/ui", () => ({
	useToast: () => ({ infoToast }),
	Skeleton: () => <div />,
}));

import CheckPermission from "@/components/custom/check-permission";
import {
	refreshEnvironment,
	refreshEnvironmentThrottled,
} from "@/lib/refresh-environment";
import { useSessionStore } from "@/store/session-store";
import { useTokenStore } from "@/store/token-store";

const env = (permissions: string[]): EnvironmentWithCompany =>
	({
		ID: 7,
		name: "Admin",
		role_id: 1,
		company_id: 2,
		role: { role: "admin", permissions: permissions.map((ID) => ({ ID })) },
		company: {
			legal_name: "L",
			trade_name: "T",
			plan_code: "pro",
			tenant_id: 3,
			tenant: { name: "t", slug: "t", enabled_features: "{}" },
		},
	}) as unknown as EnvironmentWithCompany;

beforeEach(() => {
	getMyEnvironments.mockReset();
	infoToast.mockReset();
	useTokenStore.getState().setToken("tok");
	useSessionStore.getState().clean();
	useSessionStore.getState().setEnvironment(env(["OLD"]));
	useSessionStore.getState().setPermissionsFresh(false);
});

describe("refreshEnvironment", () => {
	it("updates the stored permissions from the API", async () => {
		getMyEnvironments.mockResolvedValue({
			success: true,
			data: [env(["OLD", "READ_APPOINTMENT"])],
		});
		await refreshEnvironment();
		const s = useSessionStore.getState();
		expect(s.environment?.permissions).toEqual(["OLD", "READ_APPOINTMENT"]);
		expect(s.permissionsFresh).toBe(true);
	});

	it("keeps the persisted state when the company is missing", async () => {
		getMyEnvironments.mockResolvedValue({
			success: true,
			data: [{ ...env(["NEW"]), company: undefined }],
		});
		await refreshEnvironment();
		const s = useSessionStore.getState();
		expect(s.environment?.permissions).toEqual(["OLD"]);
		expect(s.permissionsFresh).toBe(true);
	});

	it("keeps the persisted permissions on a network failure", async () => {
		getMyEnvironments.mockRejectedValue(new Error("network"));
		await refreshEnvironment();
		const s = useSessionStore.getState();
		expect(s.environment?.permissions).toEqual(["OLD"]);
		expect(s.permissionsFresh).toBe(true);
	});

	it("clears the session when the environment is gone", async () => {
		getMyEnvironments.mockResolvedValue({ success: true, data: [] });
		const loc = { href: "/" };
		vi.stubGlobal("location", loc);
		await refreshEnvironment();
		expect(useSessionStore.getState().environment).toBeUndefined();
		expect(loc.href).toBe("/login");
		vi.unstubAllGlobals();
	});
});

describe("refreshEnvironmentThrottled", () => {
	it("refreshes at most once every 30 s", async () => {
		vi.useFakeTimers({ toFake: ["Date"] });
		vi.setSystemTime(new Date("2030-01-01T00:00:00Z"));
		getMyEnvironments.mockResolvedValue({ success: true, data: [env(["A"])] });
		refreshEnvironmentThrottled();
		await vi.waitFor(() => expect(getMyEnvironments).toHaveBeenCalledTimes(1));
		vi.setSystemTime(new Date("2030-01-01T00:00:20Z"));
		refreshEnvironmentThrottled();
		expect(getMyEnvironments).toHaveBeenCalledTimes(1);
		await new Promise((r) => setTimeout(r, 0)); // let the first request settle
		vi.setSystemTime(new Date("2030-01-01T00:00:31Z"));
		refreshEnvironmentThrottled();
		await vi.waitFor(() => expect(getMyEnvironments).toHaveBeenCalledTimes(2));
		vi.useRealTimers();
	});
});

describe("CheckPermission", () => {
	const renderGuard = () =>
		render(
			<MemoryRouter initialEntries={["/guarded"]}>
				<Routes>
					<Route path="/" element={<div>home</div>} />
					<Route
						path="/guarded"
						element={<CheckPermission permissions={["READ_APPOINTMENT"]} />}
					>
						<Route index element={<div>secret</div>} />
					</Route>
				</Routes>
			</MemoryRouter>,
		);

	it("waits for the refresh before denying, then allows", async () => {
		renderGuard();
		expect(infoToast).not.toHaveBeenCalled();
		expect(screen.queryByText("home")).toBeNull();
		useSessionStore.getState().setEnvironment(env(["READ_APPOINTMENT"]));
		await waitFor(() => expect(screen.getByText("secret")).toBeDefined());
		expect(infoToast).not.toHaveBeenCalled();
	});

	it("denies once the refresh finished without the permission", async () => {
		renderGuard();
		useSessionStore.getState().setPermissionsFresh(true);
		await waitFor(() => expect(screen.getByText("home")).toBeDefined());
		expect(infoToast).toHaveBeenCalled();
	});
});
