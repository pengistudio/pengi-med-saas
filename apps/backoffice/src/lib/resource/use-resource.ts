import React from "react";
import { useNavigate } from "react-router";
import { type ID, type Resource, resourceRoutes } from "./resource";

/** The items of a resource, with loading state and removal. */
export function useResourceList<T extends { ID: number }>(
	resource: Resource<T, never, never>,
) {
	const [items, setItems] = React.useState<T[]>([]);
	const [loading, setLoading] = React.useState(true);

	const refetch = React.useCallback(async () => {
		setLoading(true);
		const res = await resource.list();
		if (res.success) setItems(res.data ?? []);
		setLoading(false);
	}, [resource]);

	React.useEffect(() => {
		refetch();
	}, [refetch]);

	const remove = React.useCallback(
		async (id: ID) => {
			const res = await resource.remove(id);
			if (res.success) await refetch();
			return res.success;
		},
		[resource, refetch],
	);

	return { items, loading, remove, refetch };
}

/**
 * One item of a resource, for its create and edit pages. With an id it loads
 * the item and `save` updates it; without one `save` creates it. After a
 * successful save it navigates back to the resource's list.
 */
export function useResourceItem<T extends { ID: number }, C, U>(
	resource: Resource<T, C, U>,
	id?: ID,
) {
	const navigate = useNavigate();
	const [item, setItem] = React.useState<T | undefined>(undefined);
	const [loading, setLoading] = React.useState(id !== undefined);
	const [saving, setSaving] = React.useState(false);

	React.useEffect(() => {
		if (id === undefined) return;
		let cancelled = false;
		setLoading(true);
		resource.get(id).then((res) => {
			if (cancelled) return;
			if (res.success) setItem(res.data);
			setLoading(false);
		});
		return () => {
			cancelled = true;
		};
	}, [resource, id]);

	const save = React.useCallback(
		async (data: C | U) => {
			setSaving(true);
			const res =
				id === undefined
					? await resource.create(data as C)
					: await resource.update(id, data as U);
			setSaving(false);
			if (res.success) navigate(resourceRoutes(resource.name).list);
			return res.success;
		},
		[resource, id, navigate],
	);

	return { item, loading, saving, save };
}
