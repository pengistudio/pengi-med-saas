import { CheckCircle2, OctagonAlert, TriangleAlert } from "lucide-react";
import type { NavigateFunction } from "react-router";
import type { NotificationLevel } from "@/api/notification-service";
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
