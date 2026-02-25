import {
	Calendar,
	Hospital,
	LayoutDashboard,
	Settings,
	Users,
	UsersRound,
} from "lucide-react";
import type React from "react";

export interface BaseNavItem {
	label: string;
	icon: React.ComponentType<{ className?: string }>;
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
	},
	{
		icon: Users,
		label: textGet("dashboard.users"),
		href: "/users",
	},
	{
		icon: Settings,
		label: textGet("dashboard.settings"),
		href: "/settings",
	},
];
