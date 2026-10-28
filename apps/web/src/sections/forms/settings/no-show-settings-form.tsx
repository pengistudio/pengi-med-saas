import { useText } from "@pengi/shared";
import { Button, Form, FormInput } from "@pengi/ui";
import { AlertTriangle, Loader2, Save } from "lucide-react";
import { z } from "zod";
import { type AgendaSettings, NO_SHOW_MINUTES } from "@/api/settings-service";
import { SettingsSection } from "@/components/custom/settings-section";
import { FormSwitch } from "@/components/forms/form-switch";

const MINUTES_ERROR = "agenda.no_show.error.minutes";

const formSchema = z.object({
	auto_no_show: z.boolean(),
	no_show_after_minutes: z
		.number({ error: MINUTES_ERROR })
		.int(MINUTES_ERROR)
		.min(NO_SHOW_MINUTES.min, MINUTES_ERROR)
		.max(NO_SHOW_MINUTES.max, MINUTES_ERROR),
});

export type NoShowSettingsValues = z.infer<typeof formSchema>;

/**
 * The clinic's automatic no-show marking: on/off and the minutes after an
 * appointment's end. The page saves it into the tenant UI settings.
 */
export function NoShowSettingsForm({
	initialData,
	loading,
	onSubmit,
}: {
	initialData: AgendaSettings;
	loading?: boolean;
	onSubmit: (values: NoShowSettingsValues) => void;
}) {
	const { textGet } = useText();

	return (
		<Form
			schema={formSchema}
			defaultValues={{
				auto_no_show: initialData.auto_no_show,
				no_show_after_minutes: initialData.no_show_after_minutes,
			}}
			onSubmit={onSubmit}
			className="grid grid-cols-1 items-start gap-4"
		>
			{(field) => (
				<>
					<SettingsSection
						title={textGet("agenda.no_show.title")}
						description={<p>{textGet("agenda.no_show.description")}</p>}
					>
						<p
							role="note"
							className="flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-800 dark:text-amber-300"
						>
							<AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
							{textGet("agenda.no_show.warning")}
						</p>
						<FormSwitch
							field={field}
							name="auto_no_show"
							label={textGet("agenda.no_show.auto")}
							description={textGet("agenda.no_show.auto.description")}
						/>
						<FormInput
							field={field}
							name="no_show_after_minutes"
							type="number"
							min={NO_SHOW_MINUTES.min}
							max={NO_SHOW_MINUTES.max}
							label={textGet("agenda.no_show.minutes")}
							description={textGet("agenda.no_show.minutes.description")}
							disabled={!field.watch("auto_no_show")}
						/>
					</SettingsSection>
					<div className="flex justify-end">
						<Button type="submit" disabled={loading}>
							{loading ? (
								<Loader2 className="mr-2 h-4 w-4 animate-spin" />
							) : (
								<Save className="mr-2 h-4 w-4" />
							)}
							{textGet("form.save")}
						</Button>
					</div>
				</>
			)}
		</Form>
	);
}
