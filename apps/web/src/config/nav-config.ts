import {
	Calendar,
	FileKey,
	Hospital,
	Layers,
	LayoutDashboard,
	Receipt,
	UsersRound,
} from "lucide-react";
import type React from "react";
import { PERMISSIONS } from "@/lib/constants";

export interface BaseNavItem {
	label: string;
	icon: React.ComponentType<{ className?: string }>;
	permission?: string;
}

export type NavItemType =
	| (BaseNavItem & { href: string; accordionItems?: never })
	| (BaseNavItem & {
			href?: never;
			accordionItems: (BaseNavItem & { href: string })[];
	  });

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
		],
		permission: PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_MEDICAL_RECORD,
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
				label: textGet("dashboard.billing.settings"),
				href: "/billing/settings",
				icon: FileKey,
			},
			{
				label: textGet("dashboard.billing.catalog-items"),
				href: "/billing/catalog-items",
				icon: Layers,
			},
		],
		permission: PERMISSIONS.BILLING.PERMISSION_READ_BILLING,
	},
];
