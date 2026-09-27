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
import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { Plus } from "lucide-react";
import { useEffect, useState } from "react";
import { getTasks, moveTask } from "@/api/kanban-service";
import { PageHeader } from "@/components/custom/page-header";
import BoardProgress from "@/sections/kanban/board-progress";
import KanbanColumn from "@/sections/kanban/kanban-column";
import { TASK_STATUSES as STATUSES } from "@/sections/kanban/status-config";
import TaskCardContent from "@/sections/kanban/task-card-content";
import TaskFormDialog from "@/sections/kanban/task-form-dialog";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useKanbanStore } from "@/store/kanban-store";
import type { TaskStatus } from "@/types/kanban-type";

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

	// Where the dragged task sits right now (drag-over moves it between lanes).
	const dropTargetStatus = activeTask
		? tasks.find((t) => t.id === activeTask.id)?.status
		: undefined;

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
			<div className="flex h-full min-h-0 flex-col gap-5">
				<PageHeader
					title={textGet("tasks.title")}
					description={textGet("tasks.page.description")}
					actions={
						<Button onClick={() => setIsFormOpen(true)}>
							<Plus className="h-4 w-4 mr-2" />
							{textGet("tasks.task.create.title")}
						</Button>
					}
				/>

				<BoardProgress tasks={tasks} />

				<DndContext
					collisionDetection={closestCenter}
					onDragStart={handleDragStart}
					onDragOver={handleDragOver}
					onDragEnd={handleDragEnd}
					sensors={sensors}
				>
					<div className="-mx-4 grid min-h-0 flex-1 items-start auto-cols-[minmax(17rem,1fr)] grid-flow-col gap-3 overflow-x-auto px-4 pb-2 sm:mx-0 sm:px-0">
						{STATUSES.map((status) => (
							<KanbanColumn
								key={status}
								status={status}
								tasks={getTasksByStatus(status)}
								isDropTarget={dropTargetStatus === status}
								onAddTask={() => handleAddTaskInColumn(status)}
							/>
						))}
					</div>

					{/* Drag Overlay */}
					<DragOverlay dropAnimation={null}>
						{activeTask ? (
							<div className="w-72 rotate-1">
								<TaskCardContent task={activeTask} lifted />
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
