import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardHeader,
	CardTitle,
	Checkbox,
} from "@pengi/ui";
import { Plus } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import type { DashboardTask } from "@/api/clinical-service";
import { moveTask } from "@/api/kanban-service";
import { TaskDueLabel } from "@/components/custom/task-due-label";

/**
 * Open tasks of the team board, nearest due date first. Ticking one moves it
 * to done right here.
 */
export function OpenTasksCard({
	tasks: initialTasks,
	total: initialTotal,
	canComplete,
	canCreate,
}: {
	tasks: DashboardTask[];
	total: number;
	canComplete: boolean;
	canCreate: boolean;
}) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [tasks, setTasks] = React.useState(initialTasks);
	const [total, setTotal] = React.useState(initialTotal);

	const complete = async (task: DashboardTask) => {
		setTasks((current) => current.filter((t) => t.id !== task.id));
		setTotal((n) => n - 1);
		const res = await moveTask(task.id, { status: "done", position: 0 });
		if (!res.success) {
			setTasks((current) => [...current, task]);
			setTotal((n) => n + 1);
		}
	};

	return (
		<Card>
			<CardHeader className="flex flex-row items-start justify-between gap-4">
				<div className="space-y-1">
					<CardTitle className="text-base">
						{textGet("dashboard.tasks.title")}
					</CardTitle>
					{total > 0 && (
						<p className="text-sm text-muted-foreground">
							{textGet("dashboard.tasks.total", { count: total })}
						</p>
					)}
				</div>
				<Button variant="ghost" size="sm" onClick={() => navigate("/tasks")}>
					{textGet("dashboard.tasks.open_board")}
				</Button>
			</CardHeader>
			<CardContent>
				{tasks.length === 0 ? (
					<div className="flex flex-col items-start gap-3 rounded-lg border border-dashed p-6">
						<p className="text-sm text-muted-foreground">
							{textGet("dashboard.tasks.empty")}
						</p>
						{canCreate && (
							<Button
								size="sm"
								variant="outline"
								onClick={() => navigate("/tasks")}
							>
								<Plus className="mr-2 h-4 w-4" />
								{textGet("tasks.task.create.title")}
							</Button>
						)}
					</div>
				) : (
					<ul className="divide-y">
						{tasks.map((task) => (
							<li key={task.id} className="flex items-center gap-3 py-2.5">
								{canComplete && (
									<Checkbox
										aria-label={textGet("dashboard.tasks.complete")}
										checked={false}
										onCheckedChange={() => complete(task)}
									/>
								)}
								<span className="min-w-0 flex-1 truncate text-sm">
									{task.title}
								</span>
								<TaskDueLabel dueDate={task.due_date} />
							</li>
						))}
					</ul>
				)}
			</CardContent>
		</Card>
	);
}
