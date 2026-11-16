import { createHttpService } from "@pengi/shared";
import { act, screen } from "@testing-library/react";
import type { AxiosAdapter } from "axios";
import { Link, MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

const toastMocks = vi.hoisted(() => ({ error: vi.fn(), success: vi.fn() }));
vi.mock("sonner", async (importOriginal) => ({
	...(await importOriginal<typeof import("sonner")>()),
	toast: Object.assign(vi.fn(), toastMocks),
}));

import { apiWithTenant } from "@/api";
import { enableTwoFactor } from "@/api/two-factor-service";
import {
	MFARequiredBanner,
	MFARequiredGuard,
} from "@/components/features/profile/mfa-required";
import { useSessionStore } from "@/store/session-store";
import { renderWithProviders } from "@/test/test-wrapper";

const replyMFARequired: AxiosAdapter = (config) =>
	Promise.reject(
		Object.assign(new Error("403"), {
			config,
			response: {
				status: 403,
				data: {
					code: 403,
					message: "tenant.security.mfa_required",
					data: { error_code: "E-AUTH-026", error_message: "" },
				},
				config,
			},
		}),
	);

const replyOK: AxiosAdapter = (config) =>
	Promise.resolve({
		data: { code: 200, message: "ok", data: { token: "t" } },
		status: 200,
		statusText: "OK",
		headers: {},
		config,
	});

const PENDING = Symbol("pending");
const settledWithin = (p: Promise<unknown>, ms = 30) =>
	Promise.race([
		p.then(() => "settled"),
		new Promise((resolve) => setTimeout(() => resolve(PENDING), ms)),
	]);

/** Renders the guard at `path` and shows where the router ended up. */
function renderAt(path: string) {
	function Where() {
		const location = useLocation();
		return (
			<>
				<div data-testid="where">{location.pathname + location.search}</div>
				<Link to="/agenda">go</Link>
			</>
		);
	}
	return renderWithProviders(
		<MemoryRouter initialEntries={[path]}>
			<MFARequiredGuard />
			<Routes>
				<Route path="*" element={<Where />} />
			</Routes>
		</MemoryRouter>,
	);
}

describe("clinic requires two-step verification", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		useSessionStore.setState({ mfaRequired: false });
	});

	it("raises the flag on E-AUTH-026 and holds the request, so no toast fires", async () => {
		apiWithTenant.defaults.adapter = replyMFARequired;
		const service = createHttpService(apiWithTenant);

		const calls = [1, 2, 3].map(() =>
			service.get("/clinical/patients", { notifyError: true }),
		);

		for (const call of calls) {
			expect(await settledWithin(call)).toBe(PENDING);
		}
		expect(useSessionStore.getState().mfaRequired).toBe(true);
		expect(toastMocks.error).not.toHaveBeenCalled();
	});

	it("clears the flag once the user turns on two-factor", async () => {
		useSessionStore.setState({ mfaRequired: true });
		apiWithTenant.defaults.adapter = replyOK;

		const res = await enableTwoFactor("123456");

		expect(res.success).toBe(true);
		expect(useSessionStore.getState().mfaRequired).toBe(false);
	});

	it("sends the user to the profile's security section", async () => {
		useSessionStore.setState({ mfaRequired: true });
		renderAt("/clinical/patients");

		expect(await screen.findByTestId("where")).toHaveTextContent(
			"/profile?section=security",
		);
	});

	it("doesn't navigate from the profile (no loop)", async () => {
		useSessionStore.setState({ mfaRequired: true });
		renderAt("/profile?section=account");

		expect(await screen.findByTestId("where")).toHaveTextContent(
			"/profile?section=account",
		);
	});

	it("reloads on the first navigation off the profile after enrolling", async () => {
		const assign = vi.fn();
		vi.stubGlobal("location", { ...window.location, assign });
		try {
			useSessionStore.setState({ mfaRequired: true });
			renderAt("/profile?section=security");
			await screen.findByTestId("where");

			act(() => useSessionStore.setState({ mfaRequired: false }));
			expect(assign).not.toHaveBeenCalled(); // still on the profile

			act(() => screen.getByText("go").click());

			expect(assign).toHaveBeenCalledTimes(1);
			expect(assign).toHaveBeenCalledWith("/agenda");
		} finally {
			vi.unstubAllGlobals();
		}
	});

	it("leaves the user alone without the flag", async () => {
		renderAt("/clinical/patients");

		expect(await screen.findByTestId("where")).toHaveTextContent(
			"/clinical/patients",
		);
	});

	it("shows one banner only while the flag is up", () => {
		renderWithProviders(<MFARequiredBanner />);
		expect(screen.queryByRole("alert")).not.toBeInTheDocument();

		act(() => useSessionStore.setState({ mfaRequired: true }));

		expect(screen.getAllByRole("alert")).toHaveLength(1);
		expect(
			screen.getByText("*profile.mfa_required.title*"),
		).toBeInTheDocument();
	});
});
