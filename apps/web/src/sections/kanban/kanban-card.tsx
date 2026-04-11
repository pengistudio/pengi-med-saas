import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useState } from "react";

import { deleteTask } from "@/api/kanban-service";
import { cn } from "@/lib/utils";
import { useKanbanStore } from "@/store/kanban-store";
import type { Task } from "@/types/kanban-type";

import TaskCardContent from "./task-card-content";
import TaskDetailsPanel from "./task-details-panel";
import TaskFormDialog from "./task-form-dialog";

interface KanbanCardProps {
	task: Task;
}

export default function KanbanCard({ task }: KanbanCardProps) {
	const {
		attributes,
		listeners,
		setNodeRef,
		transform,
		transition,
		isDragging,
		isOver,
	} = useSortable({ id: task.id });
	const { removeTask } = useKanbanStore();
	const [isEditOpen, setIsEditOpen] = useState(false);
	const [isDetailsOpen, setIsDetailsOpen] = useState(false);

	const style = {
		transform: CSS.Transform.toString(transform),
		transition,
		opacity: isDragging ? 0.4 : 1,
	};

	const handleDelete = async () => {
		const res = await deleteTask(task.id);
		if (res.success) {
			removeTask(task.id);
		}
	};

	const handleEdit = () => {
		setIsEditOpen(true);
	};

	const handleCardClick = (_e: React.MouseEvent) => {
		// Don't open details if user is dragging
		if (isDragging) return;
		setIsDetailsOpen(true);
	};

	const handleKeyDown = (e: React.KeyboardEvent) => {
		// Open details on Enter or Space
		if (e.key === "Enter" || e.key === " ") {
			e.preventDefault();
			if (!isDragging) {
				setIsDetailsOpen(true);
			}
		}
	};

	return (
		<>
			<button
				ref={setNodeRef}
				style={style}
				className={cn(
					"w-full text-left group cursor-grab active:cursor-grabbing transition-all duration-300",
					"appearance-none bg-transparent border-none padding-0 font-inherit",
					isDragging && "shadow-2xl scale-105 opacity-40",
					isOver && "ring-2 ring-blue-400/50 ring-offset-2 rounded-xl",
				)}
				{...attributes}
				{...listeners}
				onClick={handleCardClick}
				onKeyDown={handleKeyDown}
			>
				<TaskCardContent task={task} isDragging={isDragging} />
			</button>

			<TaskDetailsPanel
				open={isDetailsOpen}
				onOpenChange={setIsDetailsOpen}
				task={task}
				onEdit={handleEdit}
				onDelete={handleDelete}
			/>

			<TaskFormDialog
				open={isEditOpen}
				onOpenChange={setIsEditOpen}
				initialTask={task}
			/>
		</>
	);
}
