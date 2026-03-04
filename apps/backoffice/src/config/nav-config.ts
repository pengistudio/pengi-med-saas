import {
	Building2,
	CreditCard,
	LayoutDashboard,
	Puzzle,
	Users,
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

export const createNavItems = (
	textGet: (key: string) => string,
): NavItemType[] => [
	{
		icon: LayoutDashboard,
		label: textGet("backoffice.nav.dashboard"),
		href: "/",
	},
	{
		icon: Building2,
		label: textGet("backoffice.nav.companies"),
		accordionItems: [
			{
				icon: Building2,
				label: textGet("backoffice.nav.companies"),
				href: "/companies",
			},
			{
				icon: CreditCard,
				label: textGet("backoffice.nav.subscriptions"),
				href: "/subscriptions",
			},
			{
				icon: CreditCard,
				label: textGet("backoffice.nav.plans"),
				href: "/plans",
			},
			{
				icon: Puzzle,
				label: textGet("backoffice.nav.features"),
				href: "/features",
			},
		],
	},
	{
		icon: Users,
		label: textGet("backoffice.nav.users"),
		href: "/users",
	},
];
