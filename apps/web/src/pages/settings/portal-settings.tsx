import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { ArrowLeft, Check, Copy } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	getPortalSettings,
	type PortalSections,
	type PortalSettings,
	updatePortalSettings,
} from "@/api/portal-settings-service";
import { PageHeader } from "@/components/custom/page-header";
import { SettingsSection } from "@/components/custom/settings-section";
import { Switch } from "@/components/ui/switch";

const SECTION_KEYS: (keyof PortalSections)[] = [
	"appointments",
	"results",
	"documents",
	"invoices",
];

const DEFAULTS: PortalSettings = {
	enabled: false,
	sections: {
		appointments: true,
		results: true,
		documents: true,
		invoices: true,
	},
};

/** `/settings/portal`: turn the patient portal on and choose what patients see. */
export default function PortalSettingsPage() {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [settings, setSettings] = React.useState<PortalSettings>(DEFAULTS);
	const [portalUrl, setPortalUrl] = React.useState("");
	const [loaded, setLoaded] = React.useState(false);
	const [saving, setSaving] = React.useState(false);
	const [copied, setCopied] = React.useState(false);

	React.useEffect(() => {
		let active = true;
		getPortalSettings().then((res) => {
			if (!active) return;
			if (res.success && res.data) {
				const { portal_url, enabled, sections } = res.data;
				setSettings({ enabled, sections });
				setPortalUrl(portal_url);
			}
			setLoaded(true);
		});
		return () => {
			active = false;
		};
	}, []);

	const handleSave = async () => {
		setSaving(true);
		const res = await updatePortalSettings(settings);
		setSaving(false);
		if (res.success && res.data) setPortalUrl(res.data.portal_url);
	};

	const handleCopy = async () => {
		await navigator.clipboard.writeText(portalUrl);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("settings.portal.title")}
				description={textGet("settings.portal.description")}
				actions={
					<Button variant="outline" onClick={() => navigate("/settings")}>
						<ArrowLeft className="mr-2 h-4 w-4" />
						{textGet("settings.portal.back")}
					</Button>
				}
			/>

			<SettingsSection
				title={textGet("settings.portal.access.title")}
				description={<p>{textGet("settings.portal.access.help")}</p>}
			>
				<div className="flex items-center justify-between gap-4">
					<label htmlFor="portal-enabled" className="text-sm font-medium">
						{textGet("settings.portal.enabled")}
					</label>
					<Switch
						id="portal-enabled"
						checked={settings.enabled}
						disabled={!loaded}
						onCheckedChange={(enabled) =>
							setSettings((s) => ({ ...s, enabled }))
						}
					/>
				</div>
				<div className="grid gap-1.5">
					<p className="text-sm font-medium">
						{textGet("settings.portal.link")}
					</p>
					<div className="flex items-center gap-2">
						<code className="min-w-0 flex-1 truncate rounded-md border bg-muted px-3 py-2 text-sm">
							{portalUrl}
						</code>
						<Button
							type="button"
							variant="outline"
							disabled={!portalUrl}
							onClick={handleCopy}
						>
							{copied ? (
								<Check className="mr-2 h-4 w-4" />
							) : (
								<Copy className="mr-2 h-4 w-4" />
							)}
							{textGet(
								copied ? "settings.portal.copied" : "settings.portal.copy",
							)}
						</Button>
					</div>
				</div>
			</SettingsSection>

			<SettingsSection
				title={textGet("settings.portal.sections.title")}
				description={<p>{textGet("settings.portal.sections.help")}</p>}
			>
				<div className="divide-y">
					{SECTION_KEYS.map((key) => (
						<div
							key={key}
							className="flex items-center justify-between gap-4 py-2.5 first:pt-0 last:pb-0"
						>
							<label htmlFor={`portal-section-${key}`} className="text-sm">
								{textGet(`settings.portal.sections.${key}`)}
							</label>
							<Switch
								id={`portal-section-${key}`}
								checked={settings.sections[key]}
								disabled={!settings.enabled}
								onCheckedChange={(value) =>
									setSettings((s) => ({
										...s,
										sections: { ...s.sections, [key]: value },
									}))
								}
							/>
						</div>
					))}
				</div>
			</SettingsSection>

			<div className="flex justify-end">
				<Button onClick={handleSave} disabled={!loaded || saving}>
					{textGet("settings.portal.save")}
				</Button>
			</div>
		</main>
	);
}
