import { create } from "zustand";

import type { Task, TaskStatus } from "@/types/kanban-type";

interface KanbanStore {
	tasks: Task[];
	activeTask?: Task;
	setTasks: (tasks: Task[]) => void;
	addTask: (task: Task) => void;
	updateTask: (task: Task) => void;
	removeTask: (id: number) => void;
	setActiveTask: (task: Task | undefined) => void;
	moveTaskToColumn: (id: number, status: TaskStatus) => void;
}

export const useKanbanStore = create<KanbanStore>((set) => ({
	tasks: [],
	activeTask: undefined,
	setTasks: (tasks) => set({ tasks }),
	addTask: (task) =>
		set((state) => ({
			tasks: [...state.tasks, task],
		})),
	updateTask: (task) =>
		set((state) => ({
			tasks: state.tasks.map((t) => (t.id === task.id ? task : t)),
		})),
	removeTask: (id) =>
		set((state) => ({
			tasks: state.tasks.filter((t) => t.id !== id),
		})),
	setActiveTask: (task) => set({ activeTask: task }),
	moveTaskToColumn: (id, status) =>
		set((state) => ({
			tasks: state.tasks.map((t) => (t.id === id ? { ...t, status } : t)),
		})),
}));
