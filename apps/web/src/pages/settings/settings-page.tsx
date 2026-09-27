import { useText } from "@pengi/shared";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
	AlertDialogTrigger,
	Button,
	Text,
	useToast,
} from "@pengi/ui";
import { AlertTriangle } from "lucide-react";
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
import { PageHeader } from "@/components/custom/page-header";
import { SettingsSection } from "@/components/custom/settings-section";
import { Switch } from "@/components/ui/switch";
import useTenantSettings from "@/hooks/use-tenant-settings";
import { cn } from "@/lib/utils";
import { KanbanSettings } from "@/sections/settings/kanban-settings";
import { DashboardLayout } from "@/sections/template/dashboard-template";

/** A labelled group of rows inside a section. */
function SettingsGroup({
	title,
	children,
}: {
	title: React.ReactNode;
	children: React.ReactNode;
}) {
	return (
		<div className="grid gap-1">
			<h3 className="pb-1 text-sm font-medium text-muted-foreground">
				{title}
			</h3>
			<div className="divide-y">{children}</div>
		</div>
	);
}

function SwitchRow({
	id,
	label,
	checked,
	onToggle,
}: {
	id: string;
	label: React.ReactNode;
	checked: boolean;
	onToggle: () => void;
}) {
	return (
		<div className="flex items-center justify-between gap-4 py-2.5">
			<label htmlFor={id} className="cursor-pointer text-sm">
				{label}
			</label>
			<Switch id={id} checked={checked} onCheckedChange={onToggle} />
		</div>
	);
}

function StatusPill({
	active,
	children,
}: {
	active: boolean;
	children: React.ReactNode;
}) {
	return (
		<span className="inline-flex w-fit items-center gap-1.5 text-xs font-medium text-muted-foreground">
			<span
				className={cn(
					"size-1.5 rounded-full",
					active ? "bg-primary" : "bg-muted-foreground/50",
				)}
			/>
			{children}
		</span>
	);
}

const SettingsPage = () => {
	const { textGet } = useText();
	const { settings, saveSettings } = useTenantSettings();
	const [hasCustomTemplate, setHasCustomTemplate] = React.useState<
		boolean | null
	>(null);
	const [templateLoading, setTemplateLoading] = React.useState(false);
	const [resetConfirmOpen, setResetConfirmOpen] = React.useState(false);
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
			errorToast(null, textGet("settings.integrations.google.error"));
			setSearchParams({});
		}
	}, [searchParams, errorToast, setSearchParams, successToast, textGet]);

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
		setResetConfirmOpen(false);
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
			<div className="grid max-w-5xl gap-8">
				<PageHeader title={<Text uuid="settings.title" />} />

				<SettingsSection
					title={<Text uuid="settings.clinical.title" />}
					description={<Text uuid="settings.clinical.description" />}
				>
					<SettingsGroup
						title={<Text uuid="settings.clinical.section.table" />}
					>
						{tableToggles.map(({ key, labelKey }) => (
							<SwitchRow
								key={key}
								id={`setting-${key}`}
								label={<Text uuid={labelKey} />}
								checked={settings.clinical[key] as boolean}
								onToggle={() => toggleClinical(key)}
							/>
						))}
					</SettingsGroup>

					<SettingsGroup title={<Text uuid="settings.clinical.section.form" />}>
						{formToggles.map(({ key, labelKey }) => (
							<SwitchRow
								key={key}
								id={`setting-${key}`}
								label={<Text uuid={labelKey} />}
								checked={settings.clinical[key] as boolean}
								onToggle={() => toggleClinical(key)}
							/>
						))}

						{/* Diagnosis system: only matters while diagnoses are on */}
						{settings.clinical.show_diagnoses && (
							<div className="grid gap-2 py-2.5">
								<div className="flex items-center justify-between gap-4">
									<span className="text-sm" id="diagnosis-system-label">
										<Text uuid="settings.clinical.diagnosis_system" />
									</span>
									<fieldset
										aria-labelledby="diagnosis-system-label"
										className="inline-flex rounded-lg bg-muted p-0.5"
									>
										{(["cie11", "cie10"] as const).map((sys) => (
											<label
												key={sys}
												className="cursor-pointer rounded-md px-3 py-1 text-sm font-medium tabular-nums text-muted-foreground transition-colors hover:text-foreground has-checked:bg-background has-checked:text-foreground has-checked:shadow-xs has-focus-visible:ring-2 has-focus-visible:ring-ring"
											>
												<input
													type="radio"
													name="diagnosis-system"
													value={sys}
													checked={settings.clinical.diagnosis_system === sys}
													onChange={() => setDiagnosisSystem(sys)}
													className="sr-only"
												/>
												{sys === "cie11" ? "CIE-11" : "CIE-10"}
											</label>
										))}
									</fieldset>
								</div>
								{settings.clinical.diagnosis_system === "cie10" && (
									<p className="flex items-start gap-1.5 text-xs text-amber-600 dark:text-amber-400">
										<AlertTriangle className="mt-px size-3.5 shrink-0" />
										<Text uuid="settings.clinical.cie10_lang_warning" />
									</p>
								)}
							</div>
						)}
					</SettingsGroup>
				</SettingsSection>

				<SettingsSection
					title={<Text uuid="settings.prescription_template.title" />}
					description={
						<Text uuid="settings.prescription_template.description" />
					}
				>
					<div className="flex flex-wrap items-center justify-between gap-3">
						<StatusPill active={!!hasCustomTemplate}>
							{hasCustomTemplate ? (
								<Text uuid="settings.prescription_template.status.custom" />
							) : (
								<Text uuid="settings.prescription_template.status.default" />
							)}
						</StatusPill>
						<div className="flex gap-2">
							<input
								ref={fileInputRef}
								type="file"
								accept=".html"
								className="hidden"
								onChange={handleUploadTemplate}
							/>
							{hasCustomTemplate && (
								<AlertDialog
									open={resetConfirmOpen}
									onOpenChange={setResetConfirmOpen}
								>
									<AlertDialogTrigger
										disabled={templateLoading}
										render={
											<Button
												variant="ghost"
												size="sm"
												className="text-destructive hover:text-destructive"
											/>
										}
									>
										<Text uuid="settings.prescription_template.reset" />
									</AlertDialogTrigger>
									<AlertDialogContent>
										<AlertDialogHeader>
											<AlertDialogTitle>
												<Text uuid="settings.prescription_template.reset_confirm.title" />
											</AlertDialogTitle>
											<AlertDialogDescription>
												<Text uuid="settings.prescription_template.reset_confirm.description" />
											</AlertDialogDescription>
										</AlertDialogHeader>
										<AlertDialogFooter>
											<AlertDialogCancel>
												<Text uuid="common.cancel" />
											</AlertDialogCancel>
											<AlertDialogAction
												variant="destructive"
												disabled={templateLoading}
												onClick={handleDeleteTemplate}
											>
												<Text uuid="settings.prescription_template.reset_confirm.action" />
											</AlertDialogAction>
										</AlertDialogFooter>
									</AlertDialogContent>
								</AlertDialog>
							)}
							<Button
								variant="outline"
								size="sm"
								disabled={templateLoading}
								onClick={() => fileInputRef.current?.click()}
							>
								<Text uuid="settings.prescription_template.upload" />
							</Button>
						</div>
					</div>
				</SettingsSection>

				<SettingsSection title={<Text uuid="settings.integrations.title" />}>
					<div className="flex flex-wrap items-start justify-between gap-4">
						<div className="grid gap-1">
							<p className="text-sm font-medium">
								<Text uuid="settings.integrations.google.title" />
							</p>
							<p className="text-sm text-muted-foreground">
								<Text uuid="settings.integrations.google.description" />
							</p>
							<StatusPill active={!!googleStatus?.connected}>
								{googleStatus?.connected ? (
									<Text uuid="settings.integrations.google.connected" />
								) : (
									<Text uuid="settings.integrations.google.disconnected" />
								)}
							</StatusPill>
						</div>

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

					{/* Steps: only while not connected */}
					{!googleStatus?.connected && (
						<div className="grid gap-3 rounded-lg bg-muted/50 p-4">
							<p className="text-sm font-medium">
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
				</SettingsSection>

				<SettingsSection title={<Text uuid="tasks.title" />}>
					<KanbanSettings />
				</SettingsSection>
			</div>
		</DashboardLayout>
	);
};

export default SettingsPage;
