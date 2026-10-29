import {
	Calendar,
	CalendarClock,
	CalendarPlus,
	ClockArrowUp,
	FlaskConical,
	Layers,
	LayoutDashboard,
	MessageCircle,
	Package,
	Receipt,
	ReceiptText,
	Settings,
	SquareActivity,
	Stethoscope,
	UserPlus,
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
	/** Off unless `feature` is explicitly true (add-on features like the patient portal). */
	featureOptIn?: boolean;
	isBottom?: boolean;
	/** Title of the sidebar section this item opens; the items after it share it. */
	section?: string;
	/** Dynamic counter the layout fills in (see `NavBadgeKey`). */
	badgeKey?: NavBadgeKey;
}

export type NavBadgeKey =
	| "pendingExamReviews"
	| "whatsappUnread"
	| "pendingAppointmentRequests";

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
	patient_portal?: boolean;
}

// Factory to create navigation items with localized labels
/** Items the user may see: role permission and tenant feature flag both hold. */
export const filterNavItems = <T extends BaseNavItem>(
	items: T[],
	checkPermission: (permissions: string[]) => boolean,
	enabledFeatures: EnabledFeatures,
): T[] =>
	items.filter(
		(item) =>
			(!item.permission || checkPermission([item.permission].flat())) &&
			(!item.feature ||
				(item.featureOptIn
					? (enabledFeatures as Record<string, boolean>)[item.feature] === true
					: (enabledFeatures as Record<string, boolean>)[item.feature] !==
						false)),
	);

// Factory to create navigation items with localized labels. Items of a section
// are contiguous; its title shows only when one of them survives the filter.
export const createNavItems = (
	textGet: (key: string) => string,
): NavItemType[] => {
	const care = textGet("nav.section.care");
	const finance = textGet("nav.section.finance");
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
			label: textGet("nav.item.appointment_requests"),
			href: "/appointment-requests",
			icon: CalendarClock,
			section: care,
			badgeKey: "pendingAppointmentRequests",
			permission:
				PERMISSIONS.PATIENT_PORTAL.PERMISSION_MANAGE_APPOINTMENT_REQUESTS,
			feature: "patient_portal",
			featureOptIn: true,
		},
		{
			label: textGet("nav.item.waitlist"),
			href: "/clinical/waitlist",
			icon: ClockArrowUp,
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
			label: textGet("nav.item.documents"),
			href: "/billing",
			icon: Receipt,
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
	"/billing/catalog-items",
];
export const MAX_BOTTOM_NAV = 4;
/** Tabs left when the create button takes the centre of the bar. */
export const MAX_BOTTOM_NAV_WITH_CREATE = 3;

/** The phone tab bar items out of the already permission-filtered nav items. */
export const pickBottomNav = (
	visible: NavItemType[],
	textGet: (key: string) => string,
	max = MAX_BOTTOM_NAV,
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
	}).slice(0, max);

/**
 * What the phone's create button offers, in priority order; filtered like the
 * nav items. Each needs the route's read permission too, or CheckPermission
 * would turn the user away.
 */
export const createQuickActions = (
	textGet: (key: string) => string,
): (BaseNavItem & { href: string })[] => [
	{
		label: textGet("clinical.medical_record.new_consultation"),
		// Without a patient_id the page asks for the patient first.
		href: "/clinical/medical-records/create",
		icon: Stethoscope,
		permission: [
			PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_PATIENT,
			PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
		],
		feature: "clinical",
	},
	{
		label: textGet("dashboard.actions.new_patient"),
		href: "/clinical/create",
		icon: UserPlus,
		permission: [
			PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_PATIENT,
			PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_PATIENT,
		],
		feature: "clinical",
	},
	{
		label: textGet("dashboard.upcoming.schedule_btn"),
		// The agenda opens its new-appointment form on ?new=1.
		href: "/clinical/appointments?new=1",
		icon: CalendarPlus,
		permission: [
			PERMISSIONS.APPOINTMENTS.PERMISSION_READ_APPOINTMENT,
			PERMISSIONS.APPOINTMENTS.PERMISSION_MANAGE_APPOINTMENT,
		],
		feature: "clinical",
	},
	{
		label: textGet("nav.quick.new_exam_order"),
		href: "/clinical/exam-orders/new",
		icon: FlaskConical,
		permission: [
			PERMISSIONS.EXAM_ORDERS.PERMISSION_READ_EXAM_ORDER,
			PERMISSIONS.EXAM_ORDERS.PERMISSION_CREATE_EXAM_ORDER,
		],
		feature: "clinical",
	},
	{
		label: textGet("dashboard.actions.new_invoice"),
		href: "/billing/create",
		icon: ReceiptText,
		permission: [
			PERMISSIONS.BILLING.PERMISSION_READ_BILLING,
			PERMISSIONS.BILLING.PERMISSION_CREATE_BILLING,
		],
		feature: "billing",
	},
];
