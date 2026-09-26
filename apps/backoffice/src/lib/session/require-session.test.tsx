import { Toaster } from "@pengi/ui";
import { render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider, useLocation } from "react-router";
import { describe, expect, it } from "vitest";
import { RequireSession } from "./require-session";
import { createSession, type SessionState } from "./session";

const noGateway = {
	login: async () => null,
	refresh: async () => null,
	logout: async () => {},
};

function LoginProbe() {
	const location = useLocation();
	return <p>login page {location.search}</p>;
}

function renderAt(path: string, state: SessionState) {
	const session = createSession(noGateway);
	session.store.setState(state);
	const router = createMemoryRouter(
		[
			{ path: "/login", element: <LoginProbe /> },
			{
				path: "/plans/edit/:id",
				element: (
					<RequireSession session={session}>
						<p>edit plan</p>
					</RequireSession>
				),
			},
		],
		{ initialEntries: [path] },
	);
	render(
		<>
			<Toaster />
			<RouterProvider router={router} />
		</>,
	);
	return session;
}

describe("RequireSession", () => {
	it("shows the page when authenticated", () => {
		renderAt("/plans/edit/3", {
			status: "authenticated",
			expired: false,
			token: "t",
		});
		expect(screen.getByText("edit plan")).toBeInTheDocument();
	});

	it("waits while the session is being restored", () => {
		renderAt("/plans/edit/3", { status: "restoring", expired: false });
		expect(screen.queryByText("edit plan")).not.toBeInTheDocument();
		expect(screen.getByText("*backoffice.common.loading*")).toBeInTheDocument();
	});

	it("sends an anonymous visitor to login, remembering where they were going", async () => {
		renderAt("/plans/edit/3", { status: "anonymous", expired: false });
		expect(
			await screen.findByText("login page ?next=%2Fplans%2Fedit%2F3"),
		).toBeInTheDocument();
		expect(
			screen.queryByText("*backoffice.session.expired.title*"),
		).not.toBeInTheDocument();
	});

	it("tells the user once when the session expired", async () => {
		renderAt("/plans/edit/3", { status: "anonymous", expired: true });
		expect(
			await screen.findByText("*backoffice.session.expired.title*"),
		).toBeInTheDocument();
		expect(
			await screen.findByText("login page ?next=%2Fplans%2Fedit%2F3"),
		).toBeInTheDocument();
		expect(
			screen.getAllByText("*backoffice.session.expired.title*"),
		).toHaveLength(1);
	});
});
