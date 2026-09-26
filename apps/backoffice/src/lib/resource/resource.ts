import type { ServiceResponse } from "@pengi/shared";

export type ID = number | string;

/**
 * A backoffice resource: a collection behind /backoffice/<path> with the usual
 * list/get/create/update/remove endpoints.
 *
 * `name` drives the conventions the backoffice pages share:
 * - i18n keys: `backoffice.<name>.{title,create,list.title,list.description,empty}`
 * - routes: `/<name>`, `/<name>/create`, `/<name>/edit/:id`
 */
export interface Resource<T extends { ID: number }, C = unknown, U = C> {
	name: string;
	list(): Promise<ServiceResponse<T[]>>;
	get(id: ID): Promise<ServiceResponse<T>>;
	create(data: C): Promise<ServiceResponse<T>>;
	update(id: ID, data: U): Promise<ServiceResponse<T>>;
	remove(id: ID): Promise<ServiceResponse<null>>;
}

export const resourceRoutes = (name: string) => ({
	list: `/${name}`,
	create: `/${name}/create`,
	edit: (id: ID) => `/${name}/edit/${id}`,
});
