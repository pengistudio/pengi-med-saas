import type {
	CreateTaskPayload,
	MoveTaskPayload,
	Task,
	TasksByStatus,
	UpdateTaskPayload,
} from "@/types/kanban-type";

import { apiWithTenant } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const kanbanService = createHttpService(apiWithTenant);

export const getTasks = async (): Promise<ServiceResponse<TasksByStatus>> => {
	return kanbanService.get<TasksByStatus>("/kanban/tasks", {
		notifyError: true,
	});
};

export const createTask = async (
	payload: CreateTaskPayload,
): Promise<ServiceResponse<Task>> => {
	return kanbanService.post<Task>("/kanban/tasks", payload, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const updateTask = async (
	id: number,
	payload: UpdateTaskPayload,
): Promise<ServiceResponse<Task>> => {
	return kanbanService.put<Task>(`/kanban/tasks/${id}`, payload, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const moveTask = async (
	id: number,
	payload: MoveTaskPayload,
): Promise<ServiceResponse<Task>> => {
	return kanbanService.put<Task>(`/kanban/tasks/${id}/move`, payload, {
		notifyError: true,
	});
};

export const deleteTask = async (
	id: number,
): Promise<ServiceResponse<null>> => {
	return kanbanService.delete<null>(`/kanban/tasks/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});
};
