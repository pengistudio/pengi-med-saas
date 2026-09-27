import { Avatar, AvatarFallback } from "@pengi/ui";
import { TaskDueLabel } from "@/components/custom/task-due-label";
import { generateTaskId } from "@/lib/task-id-generator";
import { cn } from "@/lib/utils";
import { useSessionStore } from "@/store/session-store";
import type { Task } from "@/types/kanban-type";

interface TaskCardContentProps {
	task: Task;
	/** Rendered in the drag overlay: lifted off the board. */
	lifted?: boolean;
}

function getInitials(name?: string): string {
	if (!name) return "U";
	return name
		.split(" ")
		.map((n) => n[0])
		.join("")
		.toUpperCase()
		.slice(0, 2);
}

export default function TaskCardContent({
	task,
	lifted = false,
}: TaskCardContentProps) {
	const { environment } = useSessionStore();
	const customTaskId = generateTaskId(environment?.trade_name || "", task.id);
	const isDone = task.status === "done";

	return (
		<div
			className={cn(
				"space-y-2 rounded-xl border bg-card p-3 transition-shadow",
				lifted
					? "shadow-xl ring-1 ring-primary/30"
					: "shadow-xs group-hover:shadow-md",
			)}
		>
			<h3
				className={cn(
					"line-clamp-2 text-sm font-medium leading-snug",
					isDone && "text-muted-foreground",
				)}
			>
				{task.title}
			</h3>

			{task.description && !isDone && (
				<p className="line-clamp-2 text-xs text-muted-foreground">
					{task.description}
				</p>
			)}

			<div className="flex items-center gap-2 pt-0.5">
				<span className="text-xs text-muted-foreground tabular-nums">
					{customTaskId}
				</span>
				{!isDone && <TaskDueLabel dueDate={task.due_date} />}
				{task.created_by_name && (
					<Avatar className="ml-auto size-6" title={task.created_by_name}>
						<AvatarFallback className="text-[10px] font-medium">
							{getInitials(task.created_by_name)}
						</AvatarFallback>
					</Avatar>
				)}
			</div>
		</div>
	);
}
