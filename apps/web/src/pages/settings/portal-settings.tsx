import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { ArrowLeft, Loader2 } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	DEFAULT_PORTAL_BOOKING,
	getPortalSettings,
	type PortalSettings,
	updatePortalSettings,
} from "@/api/portal-settings-service";
import { PageHeader } from "@/components/custom/page-header";
import { PortalSettingsForm } from "@/sections/forms/settings/portal-settings-form";

const DEFAULTS: PortalSettings = {
	enabled: false,
	sections: {
		appointments: true,
		results: true,
		documents: true,
		invoices: true,
	},
	booking: DEFAULT_PORTAL_BOOKING,
};

/** `/settings/portal`: turn the patient portal on, choose what patients see and how they book. */
export default function PortalSettingsPage() {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [settings, setSettings] = React.useState<PortalSettings>();
	const [portalUrl, setPortalUrl] = React.useState("");
	const [saving, setSaving] = React.useState(false);

	React.useEffect(() => {
		let active = true;
		getPortalSettings().then((res) => {
			if (!active) return;
			if (res.success && res.data) {
				const { portal_url, enabled, sections, booking } = res.data;
				setSettings({
					enabled,
					sections,
					// An API without booking yet answers without the block.
					booking: { ...DEFAULT_PORTAL_BOOKING, ...booking },
				});
				setPortalUrl(portal_url);
			} else {
				setSettings(DEFAULTS);
			}
		});
		return () => {
			active = false;
		};
	}, []);

	const handleSave = async (values: PortalSettings) => {
		setSaving(true);
		const res = await updatePortalSettings(values);
		setSaving(false);
		if (res.success && res.data) setPortalUrl(res.data.portal_url);
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

			{settings ? (
				<PortalSettingsForm
					initialData={settings}
					portalUrl={portalUrl}
					loading={saving}
					onSubmit={handleSave}
				/>
			) : (
				<div className="flex h-[40vh] items-center justify-center">
					<Loader2 className="h-8 w-8 animate-spin text-primary" />
				</div>
			)}
		</main>
	);
}
