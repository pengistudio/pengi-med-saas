import { act, renderHook, waitFor } from "@testing-library/react";
import type React from "react";
import { MemoryRouter, useLocation } from "react-router";
import { describe, expect, it } from "vitest";
import { memoryResource } from "./memory-resource";
import type { ID, Resource } from "./resource";
import { useResourceItem } from "./use-resource";

type User = { ID: number; name: string };

function renderItem(resource: Resource<User, Partial<User>>, id?: ID) {
	const wrapper = ({ children }: { children: React.ReactNode }) => (
		<MemoryRouter initialEntries={["/users/edit/1"]}>{children}</MemoryRouter>
	);
	return renderHook(
		() => ({ item: useResourceItem(resource, id), location: useLocation() }),
		{ wrapper },
	);
}

describe("useResourceItem", () => {
	it("loads the item to edit, updates it and goes back to the list", async () => {
		const mem = memoryResource<User>("users", [{ ID: 1, name: "Ana" }]);
		const { result } = renderItem(mem.resource, "1");

		await waitFor(() => expect(result.current.item.loading).toBe(false));
		expect(result.current.item.item).toEqual({ ID: 1, name: "Ana" });

		await act(() => result.current.item.save({ name: "Ana María" }));

		expect(mem.calls).toEqual([
			{ op: "update", id: "1", data: { name: "Ana María" } },
		]);
		expect(result.current.location.pathname).toBe("/users");
	});

	it("without an id, creates the item", async () => {
		const mem = memoryResource<User>("users");
		const { result } = renderItem(mem.resource);

		expect(result.current.item.loading).toBe(false);
		await act(() => result.current.item.save({ name: "Luis" }));

		expect(mem.items()).toEqual([{ ID: 1, name: "Luis" }]);
		expect(result.current.location.pathname).toBe("/users");
	});

	it("stays on the page when saving fails", async () => {
		const mem = memoryResource<User>("users");
		const { result } = renderItem(mem.resource, "404");

		await waitFor(() => expect(result.current.item.loading).toBe(false));
		let saved = true;
		await act(async () => {
			saved = await result.current.item.save({ name: "x" });
		});

		expect(saved).toBe(false);
		expect(result.current.location.pathname).toBe("/users/edit/1");
	});
});
