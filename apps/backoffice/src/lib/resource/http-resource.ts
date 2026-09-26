import { api } from "@/api";
import { createHttpService } from "@/api/fetch";
import type { Resource } from "./resource";

const httpService = createHttpService(api);
const notify = { notifySuccess: true, notifyError: true };

/**
 * The HTTP adapter of a backoffice resource at /backoffice/<name>. Writes show
 * the backend's message as a toast.
 */
export function resource<
	T extends { ID: number },
	C extends Record<string, unknown> = Record<string, unknown>,
	U extends Record<string, unknown> = C,
>(name: string, path = `/backoffice/${name}`): Resource<T, C, U> {
	return {
		name,
		list: () => httpService.get<T[]>(path),
		get: (id) => httpService.get<T>(`${path}/${id}`),
		create: (data) => httpService.post<T>(path, data, notify),
		update: (id, data) => httpService.put<T>(`${path}/${id}`, data, notify),
		remove: (id) => httpService.delete<null>(`${path}/${id}`, notify),
	};
}
