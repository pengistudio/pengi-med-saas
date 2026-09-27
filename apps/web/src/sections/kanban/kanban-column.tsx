import { useDroppable } from "@dnd-kit/core";
import {
	SortableContext,
	verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { Plus } from "lucide-react";
import { cn } from "@/lib/utils";
import type { Task, TaskStatus } from "@/types/kanban-type";

import KanbanCard from "./kanban-card";
import { taskStatusConfig } from "./status-config";

interface KanbanColumnProps {
	status: TaskStatus;
	tasks: Task[];
	/** The dragged task currently sits in this lane. */
	isDropTarget?: boolean;
	onAddTask?: () => void;
}

export default function KanbanColumn({
	status,
	tasks,
	isDropTarget = false,
	onAddTask,
}: KanbanColumnProps) {
	const { textGet } = useText();
	const { setNodeRef } = useDroppable({ id: status });
	const config = taskStatusConfig[status];
	const title = textGet(config.label);

	return (
		<section
			ref={setNodeRef}
			aria-label={title}
			className={cn(
				"flex max-h-full min-h-0 flex-col rounded-2xl bg-muted/50 ring-1 ring-transparent ring-inset transition-colors",
				isDropTarget && "bg-primary/5 ring-primary/30",
			)}
		>
			<header className="flex items-center gap-2 px-4 pt-3.5 pb-2">
				<span className={cn("size-2 rounded-full", config.dot)} />
				<h2 className="text-sm font-semibold">{title}</h2>
				<span className="text-sm text-muted-foreground tabular-nums">
					{tasks.length}
				</span>
			</header>

			<SortableContext
				items={tasks.map((t) => t.id)}
				strategy={verticalListSortingStrategy}
			>
				<div className="custom-scrollbar flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto px-2.5 pb-3">
					{tasks.map((task) => (
						<KanbanCard key={task.id} task={task} />
					))}

					{tasks.length === 0 && (
						<p className="rounded-xl border border-dashed border-border px-4 py-5 text-center text-xs text-muted-foreground">
							{textGet(config.empty)}
						</p>
					)}

					{onAddTask && (
						<Button
							variant="ghost"
							size="sm"
							onClick={onAddTask}
							className="justify-start text-muted-foreground"
						>
							<Plus />
							{textGet("tasks.column.add_task")}
						</Button>
					)}
				</div>
			</SortableContext>
		</section>
	);
}
