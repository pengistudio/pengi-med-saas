import { Text } from "@pengi/ui";
import { ShieldCheck } from "lucide-react";
import React from "react";
import { updateSecuritySettings } from "@/api/settings-service";
import { Switch } from "@/components/ui/switch";
import { selectRequireMFA, useSessionStore } from "@/store/session-store";

/**
 * The clinic's "require two-step verification" switch. The current value
 * comes with the environment (tenant.require_mfa); the service toasts the
 * result, including the refusal when the admin has no second factor yet.
 */
export function SecuritySettings() {
	const requireMFA = useSessionStore(selectRequireMFA);
	const setRequireMFA = useSessionStore((s) => s.setRequireMFA);
	const [saving, setSaving] = React.useState(false);

	async function toggle() {
		const next = !requireMFA;
		setSaving(true);
		const res = await updateSecuritySettings({ require_mfa: next });
		if (res.success) setRequireMFA(res.data.require_mfa);
		setSaving(false);
	}

	return (
		<div className="grid gap-2">
			<div className="flex items-center justify-between gap-4 py-2.5">
				<label htmlFor="setting-require-mfa" className="cursor-pointer text-sm">
					<Text uuid="settings.security.require_mfa" />
				</label>
				<Switch
					id="setting-require-mfa"
					checked={requireMFA}
					disabled={saving}
					onCheckedChange={toggle}
				/>
			</div>
			<p className="flex items-start gap-1.5 text-xs text-muted-foreground">
				<ShieldCheck className="mt-px size-3.5 shrink-0" />
				<Text uuid="settings.security.require_mfa.hint" />
			</p>
			{!requireMFA && (
				<p className="text-xs text-muted-foreground">
					<Text uuid="settings.security.require_mfa.own_hint" />
				</p>
			)}
		</div>
	);
}
