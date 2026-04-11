import type { KanbanSettings as KanbanSettingsType } from "@/api/settings-service";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { useText } from "@/hooks/use-text";
import { useTenantSettingsStore } from "@/store/tenant-settings-store";

const ARCHIVE_DELAY_OPTIONS: Array<{
	value: KanbanSettingsType["auto_archive_delay"];
	labelKey: string;
}> = [
	{ value: "never", labelKey: "kanban.settings.archive_delay.never" },
	{ value: "1_day", labelKey: "kanban.settings.archive_delay.1_day" },
	{ value: "1_week", labelKey: "kanban.settings.archive_delay.1_week" },
	{ value: "2_weeks", labelKey: "kanban.settings.archive_delay.2_weeks" },
	{ value: "1_month", labelKey: "kanban.settings.archive_delay.1_month" },
];

export function KanbanSettings() {
	const { textGet } = useText();
	const { settings, saveSettings } = useTenantSettingsStore();

	const kanbanSettings = settings.kanban || { auto_archive_delay: "never" };

	const handleArchiveDelayChange = async (
		value: KanbanSettingsType["auto_archive_delay"],
	) => {
		const newSettings = {
			...settings,
			kanban: {
				...kanbanSettings,
				auto_archive_delay: value,
			},
		};
		await saveSettings(newSettings);
	};

	return (
		<div className="space-y-4">
			<div>
				<h3 className="text-sm font-semibold mb-3">
					{textGet("kanban.settings.archive_delay.label")}
				</h3>
				<RadioGroup
					value={kanbanSettings.auto_archive_delay}
					onValueChange={handleArchiveDelayChange}
				>
					<div className="space-y-2">
						{ARCHIVE_DELAY_OPTIONS.map((option) => (
							<div key={option.value} className="flex items-center space-x-2">
								<RadioGroupItem value={option.value} id={option.value} />
								<Label
									htmlFor={option.value}
									className="font-normal cursor-pointer"
								>
									{textGet(option.labelKey)}
								</Label>
							</div>
						))}
					</div>
				</RadioGroup>
			</div>
		</div>
	);
}
