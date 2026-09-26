import type { ServiceResponse } from "@/api/fetch";
import type { ID, Resource } from "./resource";

const ok = <T>(data: T): ServiceResponse<T> => ({
	success: true,
	code: 200,
	message: "",
	data,
});

const notFound = (): ServiceResponse<never> => ({
	success: false,
	code: 404,
	message: "",
	data: { error_code: "NOT_FOUND", error_message: "" },
});

/**
 * In-memory adapter for Resource, for tests. `calls` records every write so a
 * test can assert on what was sent.
 */
export function memoryResource<T extends { ID: number }>(
	name: string,
	initial: T[] = [],
) {
	let items = [...initial];
	let nextID = Math.max(0, ...items.map((i) => i.ID)) + 1;
	const calls: {
		op: "create" | "update" | "remove";
		id?: ID;
		data?: unknown;
	}[] = [];

	const resource: Resource<T, Partial<T>, Partial<T>> = {
		name,
		list: async () => ok([...items]),
		get: async (id) => {
			const item = items.find((i) => String(i.ID) === String(id));
			return item ? ok(item) : notFound();
		},
		create: async (data) => {
			calls.push({ op: "create", data });
			const item = { ...data, ID: nextID++ } as T;
			items.push(item);
			return ok(item);
		},
		update: async (id, data) => {
			calls.push({ op: "update", id, data });
			const index = items.findIndex((i) => String(i.ID) === String(id));
			if (index < 0) return notFound();
			items[index] = { ...items[index], ...data };
			return ok(items[index]);
		},
		remove: async (id) => {
			calls.push({ op: "remove", id });
			items = items.filter((i) => String(i.ID) !== String(id));
			return ok(null);
		},
	};

	return { resource, calls, items: () => [...items] };
}
