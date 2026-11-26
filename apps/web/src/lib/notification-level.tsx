import {
	Bell,
	CalendarClock,
	CalendarPlus,
	CheckCircle2,
	ClipboardList,
	FileText,
	type LucideIcon,
	Megaphone,
	MessageCircle,
	OctagonAlert,
	TriangleAlert,
	UserPlus,
} from "lucide-react";
import type { NavigateFunction } from "react-router";
import type {
	Notification,
	NotificationLevel,
} from "@/api/notification-service";
import { cn } from "@/lib/utils";

const levelIcons = {
	success: { icon: CheckCircle2, className: "text-emerald-600" },
	warning: { icon: TriangleAlert, className: "text-amber-600" },
	critical: { icon: OctagonAlert, className: "text-destructive" },
} as const;

/**
 * Marks a notification's level with an icon. "info" (every system-generated
 * notification) shows nothing, so only what the backoffice flagged stands out.
 */
export function NotificationLevelIcon({
	level,
	className,
}: {
	level: NotificationLevel | undefined;
	className?: string;
}) {
	if (!level || level === "info") return null;
	const { icon: Icon, className: color } = levelIcons[level];
	return <Icon className={cn("h-4 w-4 shrink-0", color, className)} />;
}

/**
 * The icon of a notification in the bell and the list: WhatsApp ones show the
 * chat icon (a usage warning keeps its level icon), the rest their level.
 */
export function NotificationIcon({
	notification,
	className,
}: {
	notification: Pick<Notification, "type" | "level">;
	className?: string;
}) {
	const isWhatsApp = notification.type?.startsWith("whatsapp.");
	if (
		isWhatsApp &&
		(notification.type === "whatsapp.message" ||
			!notification.level ||
			notification.level === "info")
	) {
		return (
			<MessageCircle
				className={cn("h-4 w-4 shrink-0 text-emerald-600", className)}
			/>
		);
	}
	return (
		<NotificationLevelIcon level={notification.level} className={className} />
	);
}

/**
 * Follows a notification's action_url: in-app paths through the router,
 * https links (backoffice announcements) in a new tab.
 */
export function openNotificationLink(url: string, navigate: NavigateFunction) {
	if (url.startsWith("https://")) {
		window.open(url, "_blank", "noopener,noreferrer");
	} else {
		navigate(url);
	}
}

const typeIcons: { prefix: string; icon: LucideIcon }[] = [
	{ prefix: "portal.booking.new_patient", icon: UserPlus },
	{ prefix: "portal.booking", icon: CalendarPlus },
	{ prefix: "whatsapp.", icon: MessageCircle },
	{ prefix: "waitlist.", icon: CalendarClock },
	{ prefix: "clinical.draft", icon: FileText },
	{ prefix: "clinical.exam_order", icon: ClipboardList },
	{ prefix: "announcement", icon: Megaphone },
];

const levelTints = {
	success: "bg-emerald-500/10 text-emerald-600",
	warning: "bg-amber-500/10 text-amber-600",
	critical: "bg-destructive/10 text-destructive",
} as const;

/**
 * Round icon for the notifications page: one per notification type, tinted by
 * the level when the backoffice or a usage alert flagged it.
 */
export function NotificationAvatar({
	notification,
	className,
}: {
	notification: Pick<Notification, "type" | "level">;
	className?: string;
}) {
	const type = notification.type ?? "";
	const Icon = typeIcons.find((t) => type.startsWith(t.prefix))?.icon ?? Bell;
	const level = notification.level;
	const tint =
		level && level !== "info"
			? levelTints[level]
			: type.startsWith("whatsapp.")
				? levelTints.success
				: "bg-primary/10 text-primary";
	return (
		<span
			className={cn(
				"flex h-9 w-9 shrink-0 items-center justify-center rounded-full",
				tint,
				className,
			)}
		>
			<Icon className="h-4 w-4" />
		</span>
	);
}
