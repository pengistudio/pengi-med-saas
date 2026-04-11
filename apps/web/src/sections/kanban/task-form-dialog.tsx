import { z } from "zod";
import { createTask, updateTask as updateTaskAPI } from "@/api/kanban-service";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { FormSelect } from "@/components/forms/form-select";
import { FormTextArea } from "@/components/forms/form-textarea";
import { Button } from "@/components/ui/button";
import {
	Sheet,
	SheetBody,
	SheetContent,
	SheetDescription,
	SheetFooter,
	SheetHeader,
	SheetTitle,
} from "@/components/ui/sheet";
import { useText } from "@/hooks/use-text";
import { useKanbanStore } from "@/store/kanban-store";
import { useSessionStore } from "@/store/session-store";
import type { Task, TaskStatus } from "@/types/kanban-type";

interface TaskFormDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	initialTask?: Task;
	initialStatus?: TaskStatus;
}

const taskSchema = z.object({
	title: z.string().min(1, "Title is required"),
	description: z.string().optional(),
	status: z.enum(["todo", "in_progress", "done"]),
	due_date: z.string().optional(),
});

const statusOptions = [
	{ label: "Por Hacer", value: "todo" },
	{ label: "En Progreso", value: "in_progress" },
	{ label: "Finalizado", value: "done" },
];

export default function TaskFormDialog({
	open,
	onOpenChange,
	initialTask,
	initialStatus,
}: TaskFormDialogProps) {
	const { textGet } = useText();
	const { addTask, updateTask } = useKanbanStore();
	const { environment } = useSessionStore();

	const handleSubmit = async (data: z.infer<typeof taskSchema>) => {
		try {
			const environmentName = environment?.name || "";

			if (initialTask) {
				// Update existing task
				const res = await updateTaskAPI(initialTask.id, data);
				if (res.success) {
					updateTask(res.data);
					onOpenChange(false);
				}
			} else {
				// Create new task
				const res = await createTask({
					...data,
					created_by_name: environmentName,
				});
				if (res.success) {
					addTask(res.data);
					onOpenChange(false);
				}
			}
		} catch (error) {
			console.error("Error submitting task:", error);
		}
	};

	return (
		<Sheet open={open} onOpenChange={onOpenChange}>
			<SheetContent className="w-full sm:w-[550px] rounded-2xl flex flex-col p-0">
				<SheetHeader className="space-y-1 text-left px-4 pt-4 pb-0">
					<SheetTitle className="text-xl font-bold">
						{initialTask
							? textGet("tasks.task.edit.title")
							: textGet("tasks.task.create.title")}
					</SheetTitle>
					<SheetDescription className="text-sm">
						{initialTask
							? textGet("tasks.task.edit.description")
							: textGet("tasks.task.create.description")}
					</SheetDescription>
				</SheetHeader>

				<Form
					schema={taskSchema}
					onSubmit={handleSubmit}
					defaultValues={{
						title: initialTask?.title ?? "",
						description: initialTask?.description ?? "",
						status: initialTask?.status ?? initialStatus ?? "todo",
						due_date: initialTask?.due_date ?? "",
					}}
				>
					{(field) => (
						<>
							<SheetBody className="pb-0">
								<FormInput
									field={field}
									name="title"
									label={textGet("tasks.task.title")}
									placeholder={textGet("tasks.task.title.placeholder")}
									required
									autoFocus
								/>

								<FormTextArea
									field={field}
									name="description"
									label={textGet("tasks.task.description")}
									placeholder={textGet("tasks.task.description.placeholder")}
									description={textGet("tasks.task.description.hint")}
								/>

								<FormSelect
									field={field}
									name="status"
									label={textGet("tasks.task.status")}
									options={statusOptions}
								/>

								<FormInput
									field={field}
									name="due_date"
									label={textGet("tasks.task.due_date")}
									type="date"
								/>
							</SheetBody>

							<SheetFooter className="border-t border-border/30">
								<Button
									type="button"
									variant="outline"
									onClick={() => onOpenChange(false)}
									className="rounded-lg px-4 py-2 text-sm font-medium transition-all"
								>
									{textGet("common.cancel")}
								</Button>
								<Button
									type="submit"
									className="rounded-lg px-6 py-2 text-sm font-medium transition-all"
								>
									{textGet("common.save")}
								</Button>
							</SheetFooter>
						</>
					)}
				</Form>
			</SheetContent>
		</Sheet>
	);
}
