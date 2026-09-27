import type { TaskStatus } from "@/types/kanban-type";

export const TASK_STATUSES: TaskStatus[] = ["todo", "in_progress", "done"];

/** Label and color of each board lane; the color also paints the progress bar. */
export const taskStatusConfig: Record<
	TaskStatus,
	{ label: string; empty: string; dot: string }
> = {
	todo: {
		label: "tasks.column.todo",
		empty: "tasks.lane.empty.todo",
		dot: "bg-sky-500",
	},
	in_progress: {
		label: "tasks.column.in_progress",
		empty: "tasks.lane.empty.in_progress",
		dot: "bg-amber-500",
	},
	done: {
		label: "tasks.column.done",
		empty: "tasks.lane.empty.done",
		dot: "bg-primary",
	},
};
