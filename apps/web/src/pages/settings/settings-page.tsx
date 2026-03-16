import type {
	ClinicalSettings,
	TenantUISettings,
} from "@/api/settings-service";
import { Switch } from "@/components/ui/switch";
import { Text } from "@/components/ui/text";
import useTenantSettings from "@/hooks/use-tenant-settings";
import { cn } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const SettingsPage = () => {
	const { settings, saveSettings } = useTenantSettings();

	function toggleClinical(key: keyof ClinicalSettings) {
		saveSettings({
			...settings,
			clinical: {
				...settings.clinical,
				[key]: !settings.clinical[key],
			},
		});
	}

	function setDiagnosisSystem(
		system: TenantUISettings["clinical"]["diagnosis_system"],
	) {
		saveSettings({
			...settings,
			clinical: { ...settings.clinical, diagnosis_system: system },
		});
	}

	const tableToggles: { key: keyof ClinicalSettings; labelKey: string }[] = [
		{
			key: "show_next_appointment",
			labelKey: "settings.clinical.show_next_appointment",
		},
		{ key: "show_diagnosis", labelKey: "settings.clinical.show_diagnosis" },
		{ key: "show_medic", labelKey: "settings.clinical.show_medic" },
		{ key: "show_insurance", labelKey: "settings.clinical.show_insurance" },
	];

	const formToggles: { key: keyof ClinicalSettings; labelKey: string }[] = [
		{ key: "show_vital_signs", labelKey: "settings.clinical.show_vital_signs" },
		{ key: "show_diagnoses", labelKey: "settings.clinical.show_diagnoses" },
	];

	return (
		<DashboardLayout>
			<main className="max-w-2xl mx-auto p-4 sm:px-6 sm:py-0 grid gap-8">
				<h1 className="text-2xl font-semibold">
					<Text uuid="settings.title" />
				</h1>

				{/* Clinical module */}
				<section className="rounded-lg border bg-card p-6 grid gap-6">
					<h2 className="text-base font-medium text-muted-foreground">
						<Text uuid="settings.clinical.title" />
					</h2>

					{/* Patient table columns */}
					<div className="grid gap-3">
						<p className="text-sm font-medium">
							<Text uuid="settings.clinical.section.table" />
						</p>
						{tableToggles.map(({ key, labelKey }) => (
							<div
								key={key}
								className="flex items-center justify-between gap-4"
							>
								<span className="text-sm text-muted-foreground">
									<Text uuid={labelKey} />
								</span>
								<Switch
									checked={settings.clinical[key] as boolean}
									onCheckedChange={() => toggleClinical(key)}
								/>
							</div>
						))}
					</div>

					<div className="border-t" />

					{/* Consultation form sections */}
					<div className="grid gap-3">
						<p className="text-sm font-medium">
							<Text uuid="settings.clinical.section.form" />
						</p>
						{formToggles.map(({ key, labelKey }) => (
							<div
								key={key}
								className="flex items-center justify-between gap-4"
							>
								<span className="text-sm text-muted-foreground">
									<Text uuid={labelKey} />
								</span>
								<Switch
									checked={settings.clinical[key] as boolean}
									onCheckedChange={() => toggleClinical(key)}
								/>
							</div>
						))}

						{/* Diagnosis system selector */}
						{settings.clinical.show_diagnoses && (
							<div className="mt-2 rounded-md border p-4 grid gap-2">
								<p className="text-sm font-medium">
									<Text uuid="settings.clinical.diagnosis_system" />
								</p>
								<div className="flex gap-2">
									{(["cie11", "cie10"] as const).map((sys) => (
										<button
											key={sys}
											type="button"
											onClick={() => setDiagnosisSystem(sys)}
											className={cn(
												"flex-1 rounded-md border px-4 py-2 text-sm font-medium transition-colors",
												settings.clinical.diagnosis_system === sys
													? "bg-primary text-primary-foreground border-primary"
													: "bg-background text-muted-foreground hover:bg-muted",
											)}
										>
											{sys.toUpperCase()}
										</button>
									))}
								</div>
								{settings.clinical.diagnosis_system === "cie10" && (
									<p className="text-xs text-amber-600 dark:text-amber-400">
										⚠ <Text uuid="settings.clinical.cie10_lang_warning" />
									</p>
								)}
							</div>
						)}
					</div>
				</section>
			</main>
		</DashboardLayout>
	);
};

export default SettingsPage;
