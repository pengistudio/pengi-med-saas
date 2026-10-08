import {
	Calendar,
	CreditCard,
	FileKey,
	FileMinus,
	FilePlus,
	FlaskConical,
	Layers,
	LayoutDashboard,
	MessageCircle,
	Package,
	Receipt,
	Settings,
	ShieldCheck,
	SquareActivity,
	Users,
	UsersRound,
} from "lucide-react";
import type React from "react";
import { PERMISSIONS } from "@/lib/constants";

export interface BaseNavItem {
	label: string;
	icon: React.ComponentType<{ className?: string }>;
	/** Required permission(s); with several, the user needs all of them. */
	permission?: string | string[];
	feature?: string;
	isBottom?: boolean;
	/** Title of the sidebar section this item opens; the items after it share it. */
	section?: string;
	/** Dynamic counter the layout fills in (see `NavBadgeKey`). */
	badgeKey?: NavBadgeKey;
}

export type NavBadgeKey = "pendingExamReviews" | "whatsappUnread";

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
	whatsapp?: boolean;
}

// Factory to create navigation items with localized labels
/** Items the user may see: role permission and tenant feature flag both hold. */
export const filterNavItems = (
	items: NavItemType[],
	checkPermission: (permissions: string[]) => boolean,
	enabledFeatures: EnabledFeatures,
): NavItemType[] =>
	items.filter(
		(item) =>
			(!item.permission || checkPermission([item.permission].flat())) &&
			(!item.feature ||
				(enabledFeatures as Record<string, boolean>)[item.feature] !== false),
	);

// Factory to create navigation items with localized labels. Items of a section
// are contiguous; its title shows only when one of them survives the filter.
export const createNavItems = (
	textGet: (key: string) => string,
): NavItemType[] => {
	const care = textGet("nav.section.care");
	const finance = textGet("nav.section.finance");
	const admin = textGet("nav.section.admin");
	const appointments = {
		permission: PERMISSIONS.APPOINTMENTS.PERMISSION_READ_APPOINTMENT,
		feature: "clinical",
	};
	const billing = {
		permission: PERMISSIONS.BILLING.PERMISSION_READ_BILLING,
		feature: "billing",
	};
	return [
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
			label: textGet("nav.item.agenda"),
			href: "/clinical/appointments",
			icon: Calendar,
			section: care,
			...appointments,
		},
		{
			label: textGet("dashboard.clinical.waiting_room"),
			href: "/clinical/waiting-room",
			icon: SquareActivity,
			section: care,
			...appointments,
		},
		{
			label: textGet("dashboard.clinical.patients"),
			href: "/clinical",
			icon: UsersRound,
			section: care,
			// Front desk and doctors register patients; contador only looks them
			// up while invoicing, so READ_PATIENT alone doesn't earn the item.
			permission: [
				PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_PATIENT,
				PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_PATIENT,
			],
			feature: "clinical",
		},
		{
			icon: FlaskConical,
			label: textGet("nav.item.exams"),
			href: "/clinical/exam-orders",
			section: care,
			badgeKey: "pendingExamReviews",
			permission: PERMISSIONS.EXAM_ORDERS.PERMISSION_READ_EXAM_ORDER,
			feature: "clinical",
		},
		{
			icon: MessageCircle,
			label: textGet("nav.item.whatsapp"),
			href: "/whatsapp",
			section: care,
			badgeKey: "whatsappUnread",
			permission: PERMISSIONS.WHATSAPP.PERMISSION_USE_WHATSAPP_INBOX,
			feature: "whatsapp",
		},
		{
			label: textGet("dashboard.billing.invoices"),
			href: "/billing",
			icon: Receipt,
			section: finance,
			...billing,
		},
		{
			label: textGet("dashboard.billing.credit-notes"),
			href: "/billing/credit-notes",
			icon: FileMinus,
			section: finance,
			...billing,
		},
		{
			label: textGet("dashboard.billing.debit-notes"),
			href: "/billing/debit-notes",
			icon: FilePlus,
			section: finance,
			...billing,
		},
		{
			label: textGet("dashboard.billing.catalog-items"),
			href: "/billing/catalog-items",
			icon: Package,
			section: finance,
			...billing,
		},
		{
			label: textGet("dashboard.billing.settings"),
			href: "/billing/settings",
			icon: FileKey,
			section: finance,
			...billing,
		},
		{
			icon: Users,
			label: textGet("team.title"),
			href: "/team",
			section: admin,
			permission: PERMISSIONS.TEAM.PERMISSION_READ_TEAM,
			feature: "team",
		},
		{
			icon: ShieldCheck,
			label: textGet("audit.title"),
			href: "/audit",
			section: admin,
			permission: PERMISSIONS.AUDIT.PERMISSION_READ_AUDIT_LOG,
			// READ_AUDIT_LOG comes with the clinical module in every plan.
			feature: "clinical",
		},
		{
			icon: CreditCard,
			label: textGet("subscription.nav.title"),
			href: "/subscription",
			isBottom: true,
		},
		{
			icon: Settings,
			label: textGet("settings.title"),
			href: "/settings",
			isBottom: true,
		},
	];
};

/** Primary phone tabs, in priority order: the first MAX_BOTTOM_NAV visible ones win. */
const BOTTOM_NAV_CANDIDATES = [
	"/",
	"/clinical/appointments",
	"/clinical",
	"/clinical/exam-orders",
	"/clinical/waiting-room",
	"/billing",
	"/billing/credit-notes",
	"/billing/catalog-items",
];
export const MAX_BOTTOM_NAV = 4;

/** The phone tab bar items out of the already permission-filtered nav items. */
export const pickBottomNav = (
	visible: NavItemType[],
	textGet: (key: string) => string,
): (BaseNavItem & { href: string })[] =>
	BOTTOM_NAV_CANDIDATES.flatMap((href) => {
		const item = visible.find((i) => !i.isBottom && i.href === href);
		if (!item?.href) return [];
		return [
			{
				...item,
				href: item.href,
				label: href === "/" ? textGet("nav.item.home") : item.label,
			},
		];
	}).slice(0, MAX_BOTTOM_NAV);
