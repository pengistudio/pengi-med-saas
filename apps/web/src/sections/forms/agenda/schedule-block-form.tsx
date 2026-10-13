import { parseDateOnly, toDateOnlyString, useText } from "@pengi/shared";
import { Button, Form, FormCheckbox, FormInput } from "@pengi/ui";
import { Loader2, Save } from "lucide-react";
import { z } from "zod";
import type { BlockPayload, ScheduleBlock } from "@/api/agenda-service";
import {
	endFromInput,
	endToInput,
	isStepTime,
} from "@/components/features/agenda/agenda-utils";
import { timeToMinutes } from "@/components/features/appointments/appointment-utils";
import { FormCalendar } from "@/components/forms/form-calendar";

const blockSchema = z
	.object({
		start_date: z.date({ error: "agenda.blocks.error.date_required" }),
		end_date: z.date({ error: "agenda.blocks.error.date_required" }),
		full_day: z.boolean(),
		start_time: z.string(),
		end_time: z.string(),
		reason: z.string().trim().max(255, "agenda.blocks.error.reason_too_long"),
	})
	.superRefine((values, ctx) => {
		if (
			toDateOnlyString(values.end_date) < toDateOnlyString(values.start_date)
		) {
			ctx.addIssue({
				code: "custom",
				path: ["end_date"],
				message: "agenda.blocks.error.date_order",
			});
		}
		if (values.full_day) return;
		const start = values.start_time;
		const end = endFromInput(values.end_time);
		if (!isStepTime(start) || !isStepTime(end)) {
			ctx.addIssue({
				code: "custom",
				path: ["end_time"],
				message: "agenda.schedule.error.invalid_time",
			});
		} else if (timeToMinutes(start) >= timeToMinutes(end)) {
			ctx.addIssue({
				code: "custom",
				path: ["end_time"],
				message: "agenda.blocks.error.time_order",
			});
		}
	});

type BlockValues = z.infer<typeof blockSchema>;

/** The API body from the form: full days send both times empty. */
function toPayload(values: BlockValues): BlockPayload {
	return {
		start_date: toDateOnlyString(values.start_date),
		end_date: toDateOnlyString(values.end_date),
		start_time: values.full_day ? "" : values.start_time,
		end_time: values.full_day ? "" : endFromInput(values.end_time),
		reason: values.reason.trim(),
	};
}

/** Creates or edits a block: one or more days, whole or a time range of each. */
export function ScheduleBlockForm({
	block,
	loading,
	onSubmit,
}: {
	block?: ScheduleBlock;
	loading?: boolean;
	onSubmit: (payload: BlockPayload) => void;
}) {
	const { textGet } = useText();
	const today = new Date();
	const fullDay = block ? !block.start_time : true;
	return (
		<Form
			schema={blockSchema}
			onSubmit={(values) => onSubmit(toPayload(values))}
			defaultValues={{
				start_date: parseDateOnly(block?.start_date) ?? today,
				end_date: parseDateOnly(block?.end_date) ?? today,
				full_day: fullDay,
				start_time: block?.start_time || "08:00",
				end_time: block?.end_time ? endToInput(block.end_time) : "12:00",
				reason: block?.reason ?? "",
			}}
		>
			{(field) => (
				<div className="space-y-4">
					<div className="grid gap-4 sm:grid-cols-2">
						<FormCalendar
							field={field}
							name="start_date"
							label={textGet("agenda.blocks.field.start_date")}
						/>
						<FormCalendar
							field={field}
							name="end_date"
							label={textGet("agenda.blocks.field.end_date")}
						/>
					</div>
					<FormCheckbox
						field={field}
						name="full_day"
						label={textGet("agenda.blocks.field.full_day")}
					/>
					{!field.watch("full_day") && (
						<div className="grid grid-cols-2 gap-4">
							<FormInput
								field={field}
								name="start_time"
								type="time"
								step={300}
								label={textGet("agenda.blocks.field.start_time")}
							/>
							<FormInput
								field={field}
								name="end_time"
								type="time"
								step={300}
								label={textGet("agenda.blocks.field.end_time")}
							/>
						</div>
					)}
					<FormInput
						field={field}
						name="reason"
						label={textGet("agenda.blocks.field.reason")}
						placeholder={textGet("agenda.blocks.field.reason_placeholder")}
						isOptional
					/>
					<div className="flex justify-end">
						<Button type="submit" disabled={loading}>
							{loading ? (
								<Loader2 className="mr-2 h-4 w-4 animate-spin" />
							) : (
								<Save className="mr-2 h-4 w-4" />
							)}
							{textGet("agenda.blocks.save")}
						</Button>
					</div>
				</div>
			)}
		</Form>
	);
}
