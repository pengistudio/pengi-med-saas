import { useDroppable } from "@dnd-kit/core";
import {
	SortableContext,
	verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CheckCircle2, Clock, Plus, Zap } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";
import type { Task, TaskStatus } from "@/types/kanban-type";

import KanbanCard from "./kanban-card";

interface KanbanColumnProps {
	status: TaskStatus;
	tasks: Task[];
	totalTasks?: number;
	onAddTask?: () => void;
}

const statusLabels: Record<TaskStatus, string> = {
	todo: "tasks.column.todo",
	in_progress: "tasks.column.in_progress",
	done: "tasks.column.done",
};

const statusConfig = {
	todo: {
		icon: Clock,
		color: "border-t-blue-400",
		accentColor: "text-blue-600",
		bgLight: "bg-blue-50",
		badgeColor: "bg-blue-100 text-blue-700",
		progressColor: "bg-blue-400",
	},
	in_progress: {
		icon: Zap,
		color: "border-t-amber-400",
		accentColor: "text-amber-600",
		bgLight: "bg-amber-50",
		badgeColor: "bg-amber-100 text-amber-700",
		progressColor: "bg-amber-400",
	},
	done: {
		icon: CheckCircle2,
		color: "border-t-emerald-400",
		accentColor: "text-emerald-600",
		bgLight: "bg-emerald-50",
		badgeColor: "bg-emerald-100 text-emerald-700",
		progressColor: "bg-emerald-400",
	},
};

export default function KanbanColumn({
	status,
	tasks,
	totalTasks = 0,
	onAddTask,
}: KanbanColumnProps) {
	const { textGet } = useText();
	const { setNodeRef } = useDroppable({ id: status });
	const config = statusConfig[status];
	const Icon = config.icon;

	// Calculate progress percentage - show only in "done" column
	const completedTasks = tasks.length;
	const progressPercent =
		totalTasks > 0 ? (completedTasks / totalTasks) * 100 : 0;

	return (
		<div
			className={cn(
				"w-80 bg-white rounded-xl border-t-4 border-l border-r border-b border-slate-200 p-3 flex flex-col gap-2.5 shadow-xs hover:shadow-sm transition-all duration-300 shrink-0",
				config.color,
			)}
			ref={setNodeRef}
		>
			{/* Header */}
			<div className="space-y-2">
				{/* Title Section */}
				<div className="flex items-center justify-between gap-2">
					<div className="flex items-center gap-3 flex-1">
						<div className={cn("p-2 rounded-lg", config.bgLight)}>
							<Icon className={cn("w-5 h-5", config.accentColor)} />
						</div>
						<div className="flex-1 min-w-0">
							<h2 className="font-semibold text-base text-foreground">
								{textGet(statusLabels[status])}
							</h2>
							<p className="text-xs text-muted-foreground mt-0.5 font-medium">
								{totalTasks}{" "}
								{textGet(
									totalTasks === 1
										? "tasks.count_singular"
										: "tasks.count_plural",
								)}
							</p>
						</div>
					</div>
					<div
						className={cn(
							"px-2.5 py-1 rounded-full text-xs font-semibold",
							config.badgeColor,
						)}
					>
						{totalTasks}
					</div>
				</div>

				{/* Progress Bar - Only in "done" column */}
				{status === "done" && totalTasks > 0 && (
					<div className="space-y-2">
						<div className="flex justify-between items-center">
							<span className="text-xs font-semibold text-muted-foreground">
								{textGet("tasks.column.progress")}
							</span>
							<span className="text-xs font-bold text-muted-foreground">
								{completedTasks}/{totalTasks}
							</span>
						</div>
						<div className="w-full bg-muted/40 rounded-full h-2.5 overflow-hidden">
							<div
								className={cn(
									"h-full transition-all duration-500 rounded-full",
									config.progressColor,
								)}
								style={{ width: `${progressPercent}%` }}
							/>
						</div>
					</div>
				)}
			</div>

			{/* Tasks Container */}
			<SortableContext
				items={tasks.map((t) => t.id)}
				strategy={verticalListSortingStrategy}
			>
				<div className="flex-1 space-y-2.5 overflow-y-auto pr-2 custom-scrollbar">
					{tasks.length === 0 ? (
						<div className="flex flex-col items-center justify-center h-40 text-muted-foreground py-6">
							<div className={cn("mb-3 opacity-30", config.accentColor)}>
								<Icon className="w-12 h-12" />
							</div>
							<p className="text-sm font-semibold text-center">
								{status === "todo"
									? textGet("tasks.empty.todo.title")
									: status === "in_progress"
										? textGet("tasks.empty.in_progress.title")
										: textGet("tasks.empty.done.title")}
							</p>
							<p className="text-xs mt-2 text-center max-w-xs text-muted-foreground/70 font-light">
								{status === "todo"
									? textGet("tasks.empty.todo.description")
									: status === "in_progress"
										? textGet("tasks.empty.in_progress.description")
										: textGet("tasks.empty.done.description")}
							</p>
						</div>
					) : (
						tasks.map((task) => <KanbanCard key={task.id} task={task} />)
					)}
				</div>
			</SortableContext>

			{/* Add Card Button */}
			{onAddTask && (
				<Button
					onClick={onAddTask}
					variant="ghost"
					className="w-full justify-start text-muted-foreground hover:text-foreground hover:bg-slate-100 gap-2 text-xs font-medium transition-all duration-200 h-8"
				>
					<Plus className="h-3.5 w-3.5" />
					{textGet("tasks.column.add_card")}
				</Button>
			)}
		</div>
	);
}
