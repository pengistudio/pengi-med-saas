import { Bookmark } from "lucide-react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useText } from "@/hooks/use-text";
import { generateTaskId } from "@/lib/task-id-generator";
import { cn } from "@/lib/utils";
import { useSessionStore } from "@/store/session-store";
import type { Task } from "@/types/kanban-type";

interface TaskCardContentProps {
	task: Task;
	isDragging?: boolean;
}

const statusConfig = {
	todo: {
		label: "tasks.column.todo",
		color: "bg-blue-100 text-blue-700",
	},
	in_progress: {
		label: "tasks.column.in_progress",
		color: "bg-amber-100 text-amber-700",
	},
	done: {
		label: "tasks.column.done",
		color: "bg-emerald-100 text-emerald-700",
	},
};

const borderColorByStatus = {
	todo: "border-l-blue-400",
	in_progress: "border-l-amber-400",
	done: "border-l-emerald-400",
};

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
	isDragging = false,
}: TaskCardContentProps) {
	const { textGet } = useText();
	const { environment } = useSessionStore();

	const borderColor = borderColorByStatus[task.status] || "border-l-blue-400";
	const statusStyle = statusConfig[task.status] || statusConfig.todo;

	// Get company name for custom task ID
	const companyName = environment?.trade_name || "";
	const customTaskId = generateTaskId(companyName, task.id);

	return (
		<div
			className={cn(
				"rounded-xl border-l-4 bg-white p-3 shadow-sm space-y-2.5 transition-all duration-300",
				borderColor,
				isDragging
					? "shadow-2xl scale-105 opacity-40"
					: "hover:shadow-lg hover:-translate-y-1",
			)}
		>
			{/* Title - Prominent */}
			<h3 className="font-bold text-sm leading-tight text-foreground line-clamp-2">
				{task.title}
			</h3>

			{/* Status Badge */}
			<div
				className={cn(
					"inline-block px-2.5 py-1 rounded-md text-xs font-semibold",
					statusStyle.color,
				)}
			>
				{textGet(statusStyle.label)}
			</div>

			{/* Footer - ID and Creator */}
			<div className="flex items-center justify-between pt-1">
				<div className="flex items-center gap-1.5 text-muted-foreground min-w-0">
					<Bookmark className="h-3.5 w-3.5 flex-shrink-0" />
					<span className="text-xs font-semibold">{customTaskId}</span>
				</div>

				{/* Creator Avatar */}
				{task.created_by_name && (
					<Avatar
						className="h-5 w-5 flex-shrink-0"
						title={task.created_by_name}
					>
						<AvatarFallback className="text-xs font-semibold">
							{getInitials(task.created_by_name)}
						</AvatarFallback>
					</Avatar>
				)}
			</div>
		</div>
	);
}
