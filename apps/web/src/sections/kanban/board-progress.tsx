import { useText } from "@pengi/shared";
import { SegmentedProgress } from "@/components/custom/segmented-progress";
import type { Task } from "@/types/kanban-type";

import { TASK_STATUSES, taskStatusConfig } from "./status-config";

/** Each lane's share of the tasks, in lane order and color. */
export default function BoardProgress({ tasks }: { tasks: Task[] }) {
	const { textGet } = useText();
	const total = tasks.length;
	const done = tasks.filter((t) => t.status === "done").length;

	const summary =
		total === 0
			? textGet("tasks.board.empty")
			: textGet("tasks.board.summary", { done, total });

	return (
		<SegmentedProgress
			summary={summary}
			segments={TASK_STATUSES.map((status) => ({
				key: status,
				count: tasks.filter((t) => t.status === status).length,
				className: taskStatusConfig[status].dot,
			}))}
		/>
	);
}
