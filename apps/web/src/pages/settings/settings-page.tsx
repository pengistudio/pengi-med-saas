import React from "react";
import { useSearchParams } from "react-router";
import {
	disconnectGoogle,
	type GoogleIntegrationStatus,
	getGoogleAuthUrl,
	getGoogleIntegrationStatus,
} from "@/api/integration-service";
import type {
	ClinicalSettings,
	TenantUISettings,
} from "@/api/settings-service";
import {
	deletePrescriptionTemplate,
	getPrescriptionTemplateStatus,
	uploadPrescriptionTemplate,
} from "@/api/settings-service";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Text } from "@/components/ui/text";
import useTenantSettings from "@/hooks/use-tenant-settings";
import useToast from "@/hooks/use-toast";
import { cn } from "@/lib/utils";
import { KanbanSettings } from "@/sections/settings/kanban-settings";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const SettingsPage = () => {
	const { settings, saveSettings } = useTenantSettings();
	const [hasCustomTemplate, setHasCustomTemplate] = React.useState<
		boolean | null
	>(null);
	const [templateLoading, setTemplateLoading] = React.useState(false);
	const fileInputRef = React.useRef<HTMLInputElement>(null);

	const [googleStatus, setGoogleStatus] =
		React.useState<GoogleIntegrationStatus | null>(null);
	const [googleLoading, setGoogleLoading] = React.useState(false);
	const [searchParams, setSearchParams] = useSearchParams();
	const { successToast, errorToast } = useToast();

	React.useEffect(() => {
		getPrescriptionTemplateStatus().then((res) => {
			if (res.success && res.data) setHasCustomTemplate(res.data.has_custom);
		});
		getGoogleIntegrationStatus().then((res) => {
			if (res.success && res.data) setGoogleStatus(res.data);
		});
	}, []);

	React.useEffect(() => {
		const googleParam = searchParams.get("google");
		if (googleParam === "connected") {
			successToast("settings.integrations.google.connected");
			setGoogleStatus((prev) => ({ ...prev, connected: true }));
			setSearchParams({});
		} else if (googleParam === "error") {
			errorToast(null, "Error al conectar con Google Calendar");
			setSearchParams({});
		}
	}, [searchParams, errorToast, setSearchParams, successToast]);

	async function handleUploadTemplate(e: React.ChangeEvent<HTMLInputElement>) {
		const file = e.target.files?.[0];
		if (!file) return;
		setTemplateLoading(true);
		const res = await uploadPrescriptionTemplate(file);
		if (res.success && res.data) setHasCustomTemplate(res.data.has_custom);
		setTemplateLoading(false);
		if (fileInputRef.current) fileInputRef.current.value = "";
	}

	async function handleConnectGoogle() {
		setGoogleLoading(true);
		const res = await getGoogleAuthUrl();
		if (res.success && res.data) {
			window.location.href = res.data.url;
		}
		setGoogleLoading(false);
	}

	async function handleDisconnectGoogle() {
		setGoogleLoading(true);
		const res = await disconnectGoogle();
		if (res.success) {
			setGoogleStatus((prev) => ({ ...prev, connected: false }));
		}
		setGoogleLoading(false);
	}

	async function handleDeleteTemplate() {
		setTemplateLoading(true);
		const res = await deletePrescriptionTemplate();
		if (res.success && res.data) setHasCustomTemplate(res.data.has_custom);
		setTemplateLoading(false);
	}

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
		{
			key: "patient_age_input",
			labelKey: "settings.clinical.patient_age_input",
		},
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

				{/* Prescription template */}
				<section className="rounded-lg border bg-card p-6 grid gap-4">
					<div>
						<h2 className="text-base font-medium text-muted-foreground">
							<Text uuid="settings.prescription_template.title" />
						</h2>
						<p className="text-sm text-muted-foreground mt-1">
							<Text uuid="settings.prescription_template.description" />
						</p>
					</div>

					<div className="flex items-center gap-3">
						<span
							className={cn(
								"inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
								hasCustomTemplate
									? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
									: "bg-muted text-muted-foreground",
							)}
						>
							{hasCustomTemplate ? (
								<Text uuid="settings.prescription_template.status.custom" />
							) : (
								<Text uuid="settings.prescription_template.status.default" />
							)}
						</span>
					</div>

					<div className="flex gap-2">
						<input
							ref={fileInputRef}
							type="file"
							accept=".html"
							className="hidden"
							onChange={handleUploadTemplate}
						/>
						<Button
							variant="outline"
							size="sm"
							disabled={templateLoading}
							onClick={() => fileInputRef.current?.click()}
						>
							<Text uuid="settings.prescription_template.upload" />
						</Button>
						{hasCustomTemplate && (
							<Button
								variant="destructive"
								size="sm"
								disabled={templateLoading}
								onClick={handleDeleteTemplate}
							>
								<Text uuid="settings.prescription_template.reset" />
							</Button>
						)}
					</div>
				</section>

				{/* Google Calendar integration */}
				<section className="rounded-lg border bg-card p-6 grid gap-4">
					<h2 className="text-base font-medium text-muted-foreground">
						<Text uuid="settings.integrations.title" />
					</h2>

					{/* Header row: title + status badge + button */}
					<div className="flex items-start justify-between gap-4">
						<div className="grid gap-1">
							<p className="text-sm font-medium">
								<Text uuid="settings.integrations.google.title" />
							</p>
							<p className="text-sm text-muted-foreground">
								<Text uuid="settings.integrations.google.description" />
							</p>
							<span
								className={cn(
									"mt-1 inline-flex w-fit items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium",
									googleStatus?.connected
										? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
										: "bg-muted text-muted-foreground",
								)}
							>
								<span
									className={cn(
										"size-1.5 rounded-full",
										googleStatus?.connected
											? "bg-green-500"
											: "bg-muted-foreground/50",
									)}
								/>
								{googleStatus?.connected ? (
									<Text uuid="settings.integrations.google.connected" />
								) : (
									<Text uuid="settings.integrations.google.disconnected" />
								)}
							</span>
						</div>

						<div className="shrink-0">
							{googleStatus?.connected ? (
								<Button
									variant="outline"
									size="sm"
									disabled={googleLoading}
									onClick={handleDisconnectGoogle}
								>
									<Text uuid="settings.integrations.google.disconnect" />
								</Button>
							) : (
								<Button
									size="sm"
									disabled={googleLoading}
									onClick={handleConnectGoogle}
								>
									<Text uuid="settings.integrations.google.connect" />
								</Button>
							)}
						</div>
					</div>

					{/* Tutorial: only shown when not connected */}
					{!googleStatus?.connected && (
						<div className="rounded-md bg-muted/50 border p-4 grid gap-3">
							<p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
								<Text uuid="settings.integrations.google.tutorial.title" />
							</p>
							<ol className="grid gap-2">
								{[1, 2, 3].map((step) => (
									<li key={step} className="flex gap-3 text-sm">
										<span className="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
											{step}
										</span>
										<span className="text-muted-foreground">
											<Text
												uuid={`settings.integrations.google.tutorial.step${step}`}
											/>
										</span>
									</li>
								))}
							</ol>
						</div>
					)}
				</section>

				{/* Kanban Settings */}
				<section className="rounded-lg border bg-card p-6 grid gap-4">
					<h2 className="text-base font-medium text-muted-foreground">
						<Text uuid="kanban.settings.title" />
					</h2>
					<KanbanSettings />
				</section>
			</main>
		</DashboardLayout>
	);
};

export default SettingsPage;
