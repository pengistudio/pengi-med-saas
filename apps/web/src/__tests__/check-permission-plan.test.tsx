import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { EnvironmentWithCompany } from "@/types/user-type";

vi.mock("@pengi/ui", () => ({
	Skeleton: () => <div />,
	useToast: () => ({ infoToast: vi.fn() }),
	Button: ({ children }: { children?: React.ReactNode }) => (
		<button type="button">{children}</button>
	),
	EmptyState: ({
		title,
		illustration,
	}: {
		title?: React.ReactNode;
		illustration: string;
	}) => (
		<div data-illustration={illustration}>
			<span>{title}</span>
		</div>
	),
}));

import CheckPermission from "@/components/custom/check-permission";
import { useSessionStore } from "@/store/session-store";
import { useTokenStore } from "@/store/token-store";

const env = (
	permissions: string[],
	features: Record<string, boolean>,
): EnvironmentWithCompany =>
	({
		ID: 7,
		name: "Admin",
		role_id: 1,
		company_id: 2,
		role: { role: "admin", permissions: permissions.map((ID) => ({ ID })) },
		company: {
			legal_name: "L",
			trade_name: "T",
			plan_code: "basic",
			tenant_id: 3,
			tenant: {
				name: "t",
				slug: "t",
				enabled_features: JSON.stringify(features),
			},
		},
	}) as unknown as EnvironmentWithCompany;

const renderGuard = (guard: React.ReactElement) =>
	render(
		<MemoryRouter initialEntries={["/guarded"]}>
			<Routes>
				<Route path="/guarded" element={guard}>
					<Route index element={<div>secret</div>} />
				</Route>
			</Routes>
		</MemoryRouter>,
	);

const setup = (features: Record<string, boolean>, permissions = ["READ"]) => {
	useSessionStore.getState().setEnvironment(env(permissions, features));
	useSessionStore.getState().setPermissionsFresh(true);
};

beforeEach(() => {
	useTokenStore.getState().setToken("tok");
	useSessionStore.getState().clean();
});

describe("CheckPermission plan gating", () => {
	it("shows the not-in-plan screen when the plan excludes the feature", () => {
		setup({ cash: false });
		const { container } = renderGuard(
			<CheckPermission feature="cash" permissions={["READ"]} />,
		);
		expect(screen.queryByText("secret")).toBeNull();
		expect(screen.getByText(/error\.plan\.title/)).toBeDefined();
		expect(
			container.querySelector('[data-illustration="locked-feature"]'),
		).not.toBeNull();
	});

	it("lets the route through when the plan includes the feature", () => {
		setup({ cash: true });
		renderGuard(<CheckPermission feature="cash" permissions={["READ"]} />);
		expect(screen.getByText("secret")).toBeDefined();
	});

	it("treats an absent regular feature as included", () => {
		setup({});
		renderGuard(<CheckPermission feature="cash" permissions={["READ"]} />);
		expect(screen.getByText("secret")).toBeDefined();
	});

	it("needs an explicit true for opt-in features", () => {
		setup({});
		renderGuard(
			<CheckPermission
				feature="patient_portal"
				featureOptIn
				permissions={["READ"]}
			/>,
		);
		expect(screen.queryByText("secret")).toBeNull();
		expect(screen.getByText(/error\.plan\.title/)).toBeDefined();
	});

	it("shows no-access, not the plan screen, when only the role lacks it", () => {
		setup({ cash: true }, []);
		const { container } = renderGuard(
			<CheckPermission feature="cash" permissions={["READ"]} />,
		);
		expect(screen.getByText(/error\.permission\.title/)).toBeDefined();
		expect(
			container.querySelector('[data-illustration="no-access"]'),
		).not.toBeNull();
	});

	it("keeps working without a feature prop", () => {
		setup({ cash: false });
		renderGuard(<CheckPermission permissions={["READ"]} />);
		expect(screen.getByText("secret")).toBeDefined();
	});
});
