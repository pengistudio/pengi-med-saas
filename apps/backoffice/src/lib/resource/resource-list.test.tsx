import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type React from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { memoryResource } from "./memory-resource";
import { ResourceList } from "./resource-list";

vi.mock("@/sections/template/dashboard-template", () => ({
	DashboardLayout: ({ children }: { children: React.ReactNode }) => (
		<>{children}</>
	),
}));

type Plan = { ID: number; name: string; code: string };

const columns = [
	{ header: "backoffice.plans.col.name", cell: (p: Plan) => p.name },
	{ header: "backoffice.plans.col.code", cell: (p: Plan) => p.code },
];

function renderList(
	plans: Plan[],
	extra: Partial<React.ComponentProps<typeof ResourceList<Plan>>> = {},
) {
	const mem = memoryResource<Plan>("plans", plans);
	render(
		<MemoryRouter>
			<ResourceList
				resource={mem.resource}
				columns={columns}
				itemLabel={(p) => p.name}
				{...extra}
			/>
		</MemoryRouter>,
	);
	return mem;
}

describe("ResourceList", () => {
	it("shows one row per item, with the resource's texts", async () => {
		renderList([
			{ ID: 1, name: "Básico", code: "basic" },
			{ ID: 2, name: "Pro", code: "pro" },
		]);

		expect(await screen.findByText("Básico")).toBeInTheDocument();
		expect(screen.getByText("pro")).toBeInTheDocument();
		expect(screen.getByText("*backoffice.plans.title*")).toBeInTheDocument();
		expect(screen.getByText("*backoffice.plans.col.name*")).toBeInTheDocument();
	});

	it("shows the empty state when there are no items", async () => {
		renderList([]);

		expect(
			await screen.findByText("*backoffice.plans.empty*"),
		).toBeInTheDocument();
	});

	it("asks for confirmation before deleting, naming the item", async () => {
		const user = userEvent.setup();
		const mem = renderList([
			{ ID: 1, name: "Básico", code: "basic" },
			{ ID: 2, name: "Pro", code: "pro" },
		]);
		await screen.findByText("Pro");

		await user.click(
			screen.getAllByRole("button", { name: "*backoffice.common.delete*" })[1],
		);
		const dialog = await screen.findByRole("alertdialog");
		expect(dialog).toHaveTextContent("Pro");
		expect(mem.calls).toEqual([]);

		await user.click(
			screen.getByRole("button", {
				name: "*backoffice.common.delete.confirm*",
			}),
		);
		await waitFor(() =>
			expect(screen.queryByText("Pro")).not.toBeInTheDocument(),
		);
		expect(mem.calls).toEqual([{ op: "remove", id: 2 }]);
		expect(screen.getByText("Básico")).toBeInTheDocument();
	});

	it("does not delete when the confirmation is cancelled", async () => {
		const user = userEvent.setup();
		const mem = renderList([{ ID: 1, name: "Básico", code: "basic" }]);
		await screen.findByText("Básico");

		await user.click(
			screen.getByRole("button", { name: "*backoffice.common.delete*" }),
		);
		await user.click(
			await screen.findByRole("button", { name: "*backoffice.common.cancel*" }),
		);

		await waitFor(() =>
			expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
		);
		expect(mem.calls).toEqual([]);
		expect(screen.getByText("Básico")).toBeInTheDocument();
	});

	it("shows the page's own header and row actions", async () => {
		renderList([{ ID: 1, name: "Básico", code: "basic" }], {
			headerActions: <button type="button">register link</button>,
			rowActions: (p) => <button type="button">users of {p.code}</button>,
		});

		expect(
			await screen.findByRole("button", { name: "users of basic" }),
		).toBeInTheDocument();
		expect(
			screen.getByRole("button", { name: "register link" }),
		).toBeInTheDocument();
	});
});
