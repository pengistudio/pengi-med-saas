import { useText } from "@pengi/shared";
import { differenceInCalendarDays } from "date-fns";
import { cn } from "@/lib/utils";

/** Relative due date of a task ("Vence hoy", "En 3 días"), red once overdue. */
export function TaskDueLabel({
	dueDate,
	className,
}: {
	dueDate?: string | null;
	className?: string;
}) {
	const { textGet } = useText();
	if (!dueDate) return null;
	const days = differenceInCalendarDays(new Date(dueDate), new Date());
	const key =
		days < 0
			? "dashboard.tasks.due.overdue"
			: days === 0
				? "dashboard.tasks.due.today"
				: days === 1
					? "dashboard.tasks.due.tomorrow"
					: "dashboard.tasks.due.in_days";
	return (
		<span
			className={cn(
				"shrink-0 text-xs",
				days < 0
					? "font-medium text-destructive"
					: days <= 1
						? "font-medium text-amber-600 dark:text-amber-400"
						: "text-muted-foreground",
				className,
			)}
		>
			{textGet(key).replace("{count}", String(Math.abs(days)))}
		</span>
	);
}
