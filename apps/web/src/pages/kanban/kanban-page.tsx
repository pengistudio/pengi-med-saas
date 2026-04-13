import {
	closestCenter,
	DndContext,
	type DragEndEvent,
	type DragOverEvent,
	DragOverlay,
	type DragStartEvent,
	PointerSensor,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import { Plus } from "lucide-react";
import { useEffect, useState } from "react";
import { getTasks, moveTask } from "@/api/kanban-service";
import { Button } from "@/components/ui/button";
import { useText } from "@/hooks/use-text";
import KanbanColumn from "@/sections/kanban/kanban-column";
import TaskCardContent from "@/sections/kanban/task-card-content";
import TaskFormDialog from "@/sections/kanban/task-form-dialog";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useKanbanStore } from "@/store/kanban-store";
import type { TaskStatus } from "@/types/kanban-type";

const STATUSES: TaskStatus[] = ["todo", "in_progress", "done"];

export default function KanbanPage() {
	const { textGet } = useText();
	const { tasks, setTasks, activeTask, setActiveTask, moveTaskToColumn } =
		useKanbanStore();
	const [loading, setLoading] = useState(true);
	const [taskSnapshot, setTaskSnapshot] = useState<
		ReturnType<typeof useKanbanStore.getState>["tasks"]
	>([]);
	const [isFormOpen, setIsFormOpen] = useState(false);
	const [selectedStatus, setSelectedStatus] = useState<TaskStatus>("todo");

	const sensors = useSensors(
		useSensor(PointerSensor, {
			activationConstraint: {
				distance: 8,
			},
		}),
	);

	useEffect(() => {
		const loadTasks = async () => {
			setLoading(true);
			const res = await getTasks();
			if (res.success) {
				const allTasks = [
					...res.data.todo,
					...res.data.in_progress,
					...res.data.done,
				];
				setTasks(allTasks);
			}
			setLoading(false);
		};

		loadTasks();
	}, [setTasks]);

	const getTasksByStatus = (status: TaskStatus) => {
		return tasks.filter((t) => t.status === status);
	};

	const handleDragStart = (event: DragStartEvent) => {
		const taskId = event.active.id as number;
		const task = tasks.find((t) => t.id === taskId);
		if (task) {
			setActiveTask(task);
			setTaskSnapshot([...tasks]);
		}
	};

	const handleDragOver = (event: DragOverEvent) => {
		const { active, over } = event;
		if (!over) return;

		const taskId = active.id as number;
		const task = tasks.find((t) => t.id === taskId);
		if (!task) return;

		let newStatus: TaskStatus = task.status;

		if (
			typeof over.id === "string" &&
			STATUSES.includes(over.id as TaskStatus)
		) {
			newStatus = over.id as TaskStatus;
		} else if (typeof over.id === "number") {
			const overTask = tasks.find((t) => t.id === over.id);
			if (overTask) newStatus = overTask.status;
		}

		if (newStatus !== task.status) {
			moveTaskToColumn(taskId, newStatus);
		}
	};

	const handleDragEnd = async (event: DragEndEvent) => {
		const { active, over } = event;
		setActiveTask(undefined);

		if (!over) {
			setTasks(taskSnapshot);
			return;
		}

		const taskId = active.id as number;
		const originalTask = taskSnapshot.find((t) => t.id === taskId);
		if (!originalTask) return;

		const currentTask = tasks.find((t) => t.id === taskId);
		if (!currentTask) return;

		let newStatus: TaskStatus = currentTask.status;
		let newPosition = 0;

		if (
			typeof over.id === "string" &&
			STATUSES.includes(over.id as TaskStatus)
		) {
			newStatus = over.id as TaskStatus;
			const statusTasks = tasks
				.filter((t) => t.status === newStatus && t.id !== taskId)
				.sort((a, b) => a.position - b.position);
			newPosition =
				statusTasks.length > 0
					? statusTasks[statusTasks.length - 1].position + 1
					: 1;
		} else if (typeof over.id === "number") {
			const overTask = tasks.find((t) => t.id === over.id);
			if (!overTask) {
				setTasks(taskSnapshot);
				return;
			}
			newStatus = overTask.status;
			newPosition = overTask.position;
		} else {
			setTasks(taskSnapshot);
			return;
		}

		if (
			newStatus === originalTask.status &&
			newPosition === originalTask.position
		)
			return;

		const res = await moveTask(taskId, {
			status: newStatus,
			position: newPosition,
		});

		if (res.success) {
			const refreshRes = await getTasks();
			if (refreshRes.success) {
				const allTasks = [
					...refreshRes.data.todo,
					...refreshRes.data.in_progress,
					...refreshRes.data.done,
				];
				setTasks(allTasks);
			}
		} else {
			setTasks(taskSnapshot);
		}
	};

	const handleAddTaskInColumn = (status: TaskStatus) => {
		setSelectedStatus(status);
		setIsFormOpen(true);
	};

	if (loading) {
		return (
			<DashboardLayout>
				<div className="flex items-center justify-center h-full">
					<div className="text-center space-y-3">
						<div className="inline-flex items-center justify-center w-10 h-10">
							<div className="w-10 h-10 border-2 border-muted-foreground/20 border-t-foreground rounded-full animate-spin"></div>
						</div>
						<p className="text-sm text-muted-foreground">
							{textGet("common.loading")}
						</p>
					</div>
				</div>
			</DashboardLayout>
		);
	}

	return (
		<DashboardLayout>
			<div className="h-full flex flex-col gap-4">
				{/* Header - Refined Typography */}
				<div className="flex items-start justify-between gap-4">
					<div className="flex-1 space-y-2">
						<h1 className="text-3xl font-bold tracking-tight">
							{textGet("tasks.title")}
						</h1>
						<p className="text-sm text-muted-foreground font-light max-w-2xl">
							{textGet("tasks.page.description")}
						</p>
					</div>
					<Button
						onClick={() => setIsFormOpen(true)}
						className="gap-2 h-9 px-4 rounded-lg text-sm font-medium transition-all"
					>
						<Plus className="h-4 w-4" />
						{textGet("tasks.task.create.title")}
					</Button>
				</div>

				<DndContext
					collisionDetection={closestCenter}
					onDragStart={handleDragStart}
					onDragOver={handleDragOver}
					onDragEnd={handleDragEnd}
					sensors={sensors}
				>
					{/* Columns Container */}
					<div className="flex gap-3 flex-1 pb-4 overflow-x-auto">
						{STATUSES.map((status) => (
							<KanbanColumn
								key={status}
								status={status}
								tasks={getTasksByStatus(status)}
								totalTasks={tasks.length}
								onAddTask={() => handleAddTaskInColumn(status)}
							/>
						))}
					</div>

					{/* Drag Overlay */}
					<DragOverlay dropAnimation={null}>
						{activeTask ? (
							<div className="w-80 rotate-2 shadow-2xl">
								<TaskCardContent task={activeTask} />
							</div>
						) : null}
					</DragOverlay>
				</DndContext>

				<TaskFormDialog
					open={isFormOpen}
					onOpenChange={setIsFormOpen}
					initialStatus={selectedStatus}
				/>
			</div>
		</DashboardLayout>
	);
}
