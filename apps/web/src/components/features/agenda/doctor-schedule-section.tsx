import { useText } from "@pengi/shared";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@pengi/ui";
import { CalendarClock } from "lucide-react";
import { ScheduleBlocksList } from "./schedule-blocks-list";
import { WeeklyScheduleEditor } from "./weekly-schedule-editor";

/**
 * "Horario" of a doctor: weekly working hours and days/hours off. `scope`
 * "me" uses the current user's own profile (/doctors/me/...), a number an
 * admin's view of that doctor (/doctors/:id/...).
 */
export function DoctorScheduleSection({ scope }: { scope: "me" | number }) {
	const { textGet } = useText();
	return (
		<Card>
			<CardHeader>
				<CardTitle className="flex items-center gap-2">
					<CalendarClock className="h-5 w-5" />
					{textGet("agenda.schedule.title")}
				</CardTitle>
				<CardDescription>
					{textGet("agenda.schedule.description")}
				</CardDescription>
			</CardHeader>
			<CardContent className="grid gap-8">
				<WeeklyScheduleEditor scope={scope} />
				<section className="grid gap-3">
					<div>
						<h3 className="text-base font-semibold">
							{textGet("agenda.blocks.title")}
						</h3>
						<p className="text-sm text-muted-foreground">
							{textGet("agenda.blocks.doctor_description")}
						</p>
					</div>
					<ScheduleBlocksList scope={scope} />
				</section>
			</CardContent>
		</Card>
	);
}
