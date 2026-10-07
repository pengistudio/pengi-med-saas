import {
	CheckCircle2,
	MessageCircle,
	OctagonAlert,
	TriangleAlert,
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
