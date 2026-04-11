import {
	Calendar,
	FileKey,
	Hospital,
	Layers,
	LayoutDashboard,
	Receipt,
	Settings,
	SquareActivity,
	Users,
	UsersRound,
} from "lucide-react";
import type React from "react";
import { PERMISSIONS } from "@/lib/constants";

export interface BaseNavItem {
	label: string;
	icon: React.ComponentType<{ className?: string }>;
	permission?: string;
	feature?: string;
	isBottom?: boolean;
}

export type NavItemType =
	| (BaseNavItem & { href: string; accordionItems?: never })
	| (BaseNavItem & {
			href?: never;
			accordionItems: (BaseNavItem & { href: string })[];
	  });

export interface EnabledFeatures {
	clinical?: boolean;
	billing?: boolean;
	team?: boolean;
	kanban?: boolean;
}

// Factory to create navigation items with localized labels
export const createNavItems = (
	textGet: (key: string) => string,
): NavItemType[] => [
	{
		icon: LayoutDashboard,
		label: textGet("dashboard.title"),
		href: "/",
	},
	{
		icon: Layers,
		label: textGet("tasks.title"),
		href: "/tasks",
		permission: PERMISSIONS.KANBAN.PERMISSION_READ_KANBAN,
		feature: "kanban",
	},
	{
		icon: Hospital,
		label: textGet("dashboard.clinical"),
		accordionItems: [
			{
				label: textGet("dashboard.clinical.patients"),
				href: "/clinical",
				icon: UsersRound,
			},
			{
				label: textGet("dashboard.clinical.appointments"),
				href: "/clinical/appointments",
				icon: Calendar,
			},
			{
				label: textGet("dashboard.clinical.waiting_room"),
				href: "/clinical/waiting-room",
				icon: SquareActivity,
			},
		],
		permission: PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_MEDICAL_RECORD,
		feature: "clinical",
	},
	{
		icon: Users,
		label: textGet("team.title"),
		href: "/team",
		feature: "team",
	},
	{
		icon: Settings,
		label: textGet("settings.title"),
		href: "/settings",
		isBottom: true,
	},
	{
		icon: Receipt,
		label: textGet("dashboard.billing"),
		accordionItems: [
			{
				label: textGet("dashboard.billing.invoices"),
				href: "/billing",
				icon: Receipt,
			},
			{
				label: textGet("dashboard.billing.catalog-items"),
				href: "/billing/catalog-items",
				icon: Layers,
			},
			{
				label: textGet("dashboard.billing.settings"),
				href: "/billing/settings",
				icon: FileKey,
			},
		],
		permission: PERMISSIONS.BILLING.PERMISSION_READ_BILLING,
		feature: "billing",
	},
];
