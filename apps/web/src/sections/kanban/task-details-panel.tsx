import { CheckCircle2, Clock, Edit2, Trash2, Zap } from "lucide-react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { useText } from "@/hooks/use-text";
import { generateTaskId } from "@/lib/task-id-generator";
import { cn } from "@/lib/utils";
import { useSessionStore } from "@/store/session-store";
import type { Task } from "@/types/kanban-type";

interface TaskDetailsPanelProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	task?: Task;
	onEdit?: () => void;
	onDelete?: () => void;
}

const statusConfig = {
	todo: {
		icon: Clock,
		label: "tasks.column.todo",
		borderColor: "border-l-blue-500",
		badgeColor: "bg-blue-50 text-blue-700",
		accentColor: "text-blue-600",
	},
	in_progress: {
		icon: Zap,
		label: "tasks.column.in_progress",
		borderColor: "border-l-amber-500",
		badgeColor: "bg-amber-50 text-amber-700",
		accentColor: "text-amber-600",
	},
	done: {
		icon: CheckCircle2,
		label: "tasks.column.done",
		borderColor: "border-l-emerald-500",
		badgeColor: "bg-emerald-50 text-emerald-700",
		accentColor: "text-emerald-600",
	},
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

function formatDate(dateString?: string | null): string {
	if (!dateString) return "—";
	try {
		const date = new Date(dateString);
		return date.toLocaleDateString("es-ES", {
			year: "numeric",
			month: "long",
			day: "numeric",
		});
	} catch {
		return "—";
	}
}

export default function TaskDetailsPanel({
	open,
	onOpenChange,
	task,
	onEdit,
	onDelete,
}: TaskDetailsPanelProps) {
	const { textGet } = useText();
	const { environment } = useSessionStore();

	if (!task) return null;

	const status = statusConfig[task.status];
	const StatusIcon = status.icon;

	const companyName = environment?.trade_name || "";

	const customTaskId = generateTaskId(companyName, task.id);

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent
				className={cn(
					"max-w-2xl rounded-xl p-0 shadow-xl border-0",
					"overflow-hidden",
				)}
			>
				{/* Status color bar on the left */}
				<div
					className="absolute left-0 top-0 bottom-0 w-1"
					style={{
						background:
							task.status === "todo"
								? "#3b82f6"
								: task.status === "in_progress"
									? "#f59e0b"
									: "#10b981",
					}}
				/>

				<div className="pl-4 pr-4 pt-5 pb-5 flex flex-col gap-5 max-h-[85vh] overflow-y-auto">
					{/* Header Section */}
					<div className="space-y-3">
						{/* Title */}
						<h2 className="text-2xl font-bold text-foreground leading-tight">
							{task.title}
						</h2>

						{/* Status Badge */}
						<div className="flex items-center gap-2 w-fit">
							<div
								className={cn(
									"inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-semibold",
									status.badgeColor,
								)}
							>
								<StatusIcon className="h-3.5 w-3.5" />
								<span>{textGet(status.label)}</span>
							</div>
						</div>
					</div>

					{/* Divider */}
					<div className="h-px bg-border/40" />

					{/* Main Content */}
					<div className="space-y-5">
						{/* Description */}
						{task.description && (
							<div className="space-y-2">
								<h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
									{textGet("tasks.task.description")}
								</h3>
								<p className="text-sm text-foreground leading-relaxed whitespace-pre-wrap">
									{task.description}
								</p>
							</div>
						)}

						{/* Metadata Grid */}
						<div className="space-y-4">
							{/* Due Date */}
							{task.due_date && (
								<div className="space-y-1.5">
									<h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
										{textGet("tasks.task.due_date")}
									</h4>
									<p className="text-sm font-medium text-foreground">
										{formatDate(task.due_date)}
									</p>
								</div>
							)}

							{/* Created At & Creator in 2-column */}
							<div className="grid grid-cols-2 gap-6">
								<div className="space-y-1.5">
									<h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
										{textGet("tasks.task.created_at")}
									</h4>
									<p className="text-sm font-medium text-foreground">
										{formatDate(task.created_at)}
									</p>
								</div>

								{/* Creator */}
								{task.created_by_name && (
									<div className="space-y-1.5">
										<h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
											{textGet("tasks.task.created_by")}
										</h4>
										<div className="flex items-center gap-2.5">
											<Avatar className="h-6 w-6">
												<AvatarFallback className="text-xs font-semibold bg-muted">
													{getInitials(task.created_by_name)}
												</AvatarFallback>
											</Avatar>
											<p className="text-sm font-medium text-foreground line-clamp-1">
												{task.created_by_name}
											</p>
										</div>
									</div>
								)}
							</div>

							{/* Updated At */}
							<div className="space-y-1.5">
								<h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
									{textGet("tasks.task.updated_at")}
								</h4>
								<p className="text-sm font-medium text-foreground">
									{formatDate(task.updated_at)}
								</p>
							</div>
						</div>
					</div>

					{/* Footer - ID and Actions */}
					<div className="flex items-center justify-between gap-3 pt-2 mt-auto">
						<span className="text-xs font-mono font-semibold text-muted-foreground">
							{customTaskId}
						</span>
						<div className="flex gap-2.5">
							{onDelete && (
								<Button
									variant="outline"
									size="sm"
									onClick={() => {
										onDelete();
										onOpenChange(false);
									}}
									className="gap-2 text-red-600 hover:text-red-700 hover:bg-red-50/80 border-red-200/50"
								>
									<Trash2 className="h-4 w-4" />
									{textGet("common.delete")}
								</Button>
							)}
							{onEdit && (
								<Button
									size="sm"
									onClick={() => {
										onEdit();
										onOpenChange(false);
									}}
									className="gap-2"
								>
									<Edit2 className="h-4 w-4" />
									{textGet("common.edit")}
								</Button>
							)}
						</div>
					</div>
				</div>
			</DialogContent>
		</Dialog>
	);
}
