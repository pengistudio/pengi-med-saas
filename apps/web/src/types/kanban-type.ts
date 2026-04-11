export type TaskStatus = "todo" | "in_progress" | "done";

export interface Task {
	id: number;
	tenant_id: number;
	title: string;
	description: string;
	status: TaskStatus;
	position: number;
	due_date?: string | null;
	created_by_name?: string;
	created_at: string;
	updated_at: string;
}

export type CreateTaskPayload = {
	title: string;
	description?: string;
	status?: TaskStatus;
	due_date?: string;
	created_by_name?: string;
};

export type UpdateTaskPayload = {
	title?: string;
	description?: string;
	status?: TaskStatus;
	due_date?: string;
};

export type MoveTaskPayload = {
	status: TaskStatus;
	position: number;
};

export type TasksByStatus = {
	todo: Task[];
	in_progress: Task[];
	done: Task[];
};
