import { Button, Text } from "@pengi/ui";
import { useMemo } from "react";
import { useNavigate } from "react-router";
import { SettingsSection } from "@/components/custom/settings-section";
import {
	type EnabledFeatures,
	filterNavItems,
	type NavItemType,
} from "@/config/nav-config";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { selectEnvironment, useSessionStore } from "@/store/session-store";

interface HubLink {
	titleKey: string;
	href: string;
	permission?: string;
	feature?: string;
}

interface HubGroup {
	key: string;
	titleKey: string;
	descriptionKey: string;
	links: HubLink[];
}

// Same permission and feature checks the sidebar items had before moving here.
const GROUPS: HubGroup[] = [
	{
		key: "billing",
		titleKey: "settings.hub.billing.title",
		descriptionKey: "settings.hub.billing.description",
		links: [
			{
				titleKey: "dashboard.billing.settings",
				href: "/billing/settings",
				permission: PERMISSIONS.BILLING.PERMISSION_READ_BILLING,
				feature: "billing",
			},
		],
	},
	{
		key: "team",
		titleKey: "settings.hub.team.title",
		descriptionKey: "settings.hub.team.description",
		links: [
			{
				titleKey: "team.title",
				href: "/team",
				permission: PERMISSIONS.TEAM.PERMISSION_READ_TEAM,
				feature: "team",
			},
			{
				titleKey: "doctors.title",
				href: "/doctors",
				permission: PERMISSIONS.DOCTORS.PERMISSION_MANAGE_DOCTORS,
				feature: "clinical",
			},
			{
				titleKey: "audit.title",
				href: "/audit",
				permission: PERMISSIONS.AUDIT.PERMISSION_READ_AUDIT_LOG,
				feature: "clinical",
			},
		],
	},
	{
		key: "plan",
		titleKey: "settings.hub.plan.title",
		descriptionKey: "settings.hub.plan.description",
		links: [{ titleKey: "subscription.nav.title", href: "/subscription" }],
	},
];

const ALL_ENABLED: EnabledFeatures = {
	clinical: true,
	billing: true,
	team: true,
};

/** Admin entries that used to live in the sidebar, grouped as settings sections. */
export function AdminLinks() {
	const navigate = useNavigate();
	const { checkPermission } = usePermission();
	const rawFeatures = useSessionStore(selectEnvironment)?.enabled_features;
	const enabledFeatures: EnabledFeatures = useMemo(() => {
		if (!rawFeatures) return ALL_ENABLED;
		try {
			return JSON.parse(rawFeatures);
		} catch {
			return ALL_ENABLED;
		}
	}, [rawFeatures]);

	return GROUPS.map((group) => {
		// filterNavItems expects nav items; only href/permission/feature matter here.
		const visible = filterNavItems(
			group.links.map((l) => ({
				...l,
				label: l.titleKey,
			})) as unknown as NavItemType[],
			checkPermission,
			enabledFeatures,
		) as unknown as (HubLink & { label: string })[];
		if (visible.length === 0) return null;
		return (
			<SettingsSection
				key={group.key}
				title={<Text uuid={group.titleKey} />}
				description={<Text uuid={group.descriptionKey} />}
			>
				<div className="divide-y">
					{visible.map((link) => (
						<div
							key={link.href}
							className="flex flex-wrap items-center justify-between gap-4 py-2.5 first:pt-0 last:pb-0"
						>
							<p className="text-sm font-medium">
								<Text uuid={link.titleKey} />
							</p>
							<Button variant="outline" onClick={() => navigate(link.href)}>
								<Text uuid="settings.hub.open" />
							</Button>
						</div>
					))}
				</div>
			</SettingsSection>
		);
	});
}
