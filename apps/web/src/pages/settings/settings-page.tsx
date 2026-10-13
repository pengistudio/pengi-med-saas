import { useText } from "@pengi/shared";
import { Button, Text, useToast } from "@pengi/ui";
import { AlertTriangle, CalendarCog, FlaskConical } from "lucide-react";
import React from "react";
import { useNavigate, useSearchParams } from "react-router";
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
import { PageHeader } from "@/components/custom/page-header";
import { SettingsSection } from "@/components/custom/settings-section";
import { Switch } from "@/components/ui/switch";
import usePermission from "@/hooks/use-permission";
import useTenantSettings from "@/hooks/use-tenant-settings";
import { PERMISSIONS } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { DocumentTemplatesSettings } from "@/sections/settings/document-templates-settings";
import { KanbanSettings } from "@/sections/settings/kanban-settings";
import { WhatsAppSettings } from "@/sections/settings/whatsapp-settings";

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
	const { checkPermission } = usePermission();
	const canManageTemplates = checkPermission([
		PERMISSIONS.DOCUMENTS.PERMISSION_MANAGE_DOCUMENT_TEMPLATES,
	]);
	const navigate = useNavigate();
	const canManageExamCatalog = checkPermission([
		PERMISSIONS.EXAM_ORDERS.PERMISSION_MANAGE_EXAM_CATALOG,
	]);
	const canManageAgenda = checkPermission([
		PERMISSIONS.DOCTORS.PERMISSION_MANAGE_DOCTORS,
	]);
	const canManageWhatsApp = checkPermission([
		PERMISSIONS.WHATSAPP.PERMISSION_MANAGE_WHATSAPP,
	]);
	const [searchParams, setSearchParams] = useSearchParams();
	// Returning from the OAuth flow (?google=connected, a fresh page load) shows
	// the integration as connected until the real status arrives.
	const [googleStatus, setGoogleStatus] =
		React.useState<GoogleIntegrationStatus | null>(() =>
			searchParams.get("google") === "connected" ? { connected: true } : null,
		);
	const [googleLoading, setGoogleLoading] = React.useState(false);
	const { successToast, errorToast } = useToast();

	React.useEffect(() => {
		getGoogleIntegrationStatus().then((res) => {
			if (res.success && res.data) setGoogleStatus(res.data);
		});
	}, []);

	React.useEffect(() => {
		const googleParam = searchParams.get("google");
		if (googleParam === "connected") {
			successToast("settings.integrations.google.connected");
			setSearchParams({});
		} else if (googleParam === "error") {
			errorToast(null, textGet("settings.integrations.google.error"));
			setSearchParams({});
		}
	}, [searchParams, errorToast, setSearchParams, successToast, textGet]);

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
		<div className="grid max-w-5xl gap-8">
			<PageHeader title={<Text uuid="settings.title" />} />

			<SettingsSection
				title={<Text uuid="settings.clinical.title" />}
				description={<Text uuid="settings.clinical.description" />}
			>
				<SettingsGroup title={<Text uuid="settings.clinical.section.table" />}>
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

			{canManageTemplates && (
				<SettingsSection
					title={<Text uuid="settings.document_templates.title" />}
					description={<Text uuid="settings.document_templates.description" />}
				>
					<DocumentTemplatesSettings />
				</SettingsSection>
			)}

			{canManageExamCatalog && (
				<SettingsSection
					title={<Text uuid="clinical.exam_catalog.title" />}
					description={
						<Text uuid="clinical.exam_catalog.settings.description" />
					}
				>
					<div className="flex flex-wrap items-center justify-between gap-4">
						<p className="text-sm text-muted-foreground">
							<Text uuid="clinical.exam_catalog.settings.hint" />
						</p>
						<Button
							variant="outline"
							onClick={() => navigate("/settings/exam-catalog")}
						>
							<FlaskConical className="mr-2 h-4 w-4" />
							<Text uuid="clinical.exam_catalog.settings.open" />
						</Button>
					</div>
				</SettingsSection>
			)}

			{canManageAgenda && (
				<SettingsSection
					title={<Text uuid="agenda.settings.title" />}
					description={<Text uuid="agenda.settings.section_description" />}
				>
					<div className="flex flex-wrap items-center justify-between gap-4">
						<p className="text-sm text-muted-foreground">
							<Text uuid="agenda.settings.hint" />
						</p>
						<Button
							variant="outline"
							onClick={() => navigate("/settings/agenda")}
						>
							<CalendarCog className="mr-2 h-4 w-4" />
							<Text uuid="agenda.settings.open" />
						</Button>
					</div>
				</SettingsSection>
			)}

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

				{canManageWhatsApp && <WhatsAppSettings />}
			</SettingsSection>

			<SettingsSection title={<Text uuid="tasks.title" />}>
				<KanbanSettings />
			</SettingsSection>
		</div>
	);
};

export default SettingsPage;
