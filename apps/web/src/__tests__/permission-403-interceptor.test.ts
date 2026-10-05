import type { AxiosAdapter } from "axios";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const throttled = vi.hoisted(() => vi.fn());
vi.mock("@/lib/refresh-environment", () => ({
	refreshEnvironmentThrottled: throttled,
}));

import { apiWithTenant } from "@/api";

const reply403 =
	(errorCode: string): AxiosAdapter =>
	(config) =>
		Promise.reject(
			Object.assign(new Error("403"), {
				config,
				response: {
					status: 403,
					data: { data: { error_code: errorCode } },
					config,
				},
			}),
		);

const hit = async (errorCode: string) => {
	apiWithTenant.defaults.adapter = reply403(errorCode);
	await apiWithTenant.get("/x").catch(() => {});
	// the interceptor loads the module through a dynamic import
	await vi.dynamicImportSettled();
};

describe("403 permission interceptor", () => {
	beforeEach(() => throttled.mockClear());
	afterEach(() => vi.useRealTimers());

	it.each(["E-PERM-002", "E-PERM-003"])("refreshes on %s", async (code) => {
		await hit(code);
		expect(throttled).toHaveBeenCalledTimes(1);
	});

	it("ignores other error codes", async () => {
		await hit("E-BO-001");
		expect(throttled).not.toHaveBeenCalled();
	});
});
