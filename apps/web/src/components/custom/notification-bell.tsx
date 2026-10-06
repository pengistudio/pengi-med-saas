import { useText } from "@pengi/shared";
import {
	Badge,
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@pengi/ui";
import { Bell } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router";
import {
	markAllNotificationsAsRead,
	markNotificationAsRead,
	type Notification,
} from "@/api/notification-service";
import {
	NotificationIcon,
	openNotificationLink,
} from "@/lib/notification-level";
import { getNotificationText } from "@/lib/notification-text";
import { cn } from "@/lib/utils";
import { useNotificationStore } from "@/store/notification-store";

const UNREAD_BADGE_MAX = 9;

const NotificationBell = () => {
	const { textGet, formatRelative } = useText();
	const navigate = useNavigate();
	const notifications = useNotificationStore((s) => s.notifications);
	const unreadCount = useNotificationStore((s) => s.unreadCount);
	const markReadLocally = useNotificationStore((s) => s.markReadLocally);
	const markAllReadLocally = useNotificationStore((s) => s.markAllReadLocally);

	// Ring the bell and pop the badge only when the count goes up. The count
	// is restored from sessionStorage, so a reload doesn't count as an arrival.
	const [seenCount, setSeenCount] = useState(unreadCount);
	const [arrivals, setArrivals] = useState(0);
	if (unreadCount !== seenCount) {
		setSeenCount(unreadCount);
		if (unreadCount > seenCount) setArrivals((n) => n + 1);
	}

	// The store only holds unread notifications (see useNotificationsPoll).
	const handleSelect = (notification: Notification) => {
		markReadLocally(notification.ID);
		markNotificationAsRead(notification.ID);
		if (notification.action_url) {
			openNotificationLink(notification.action_url, navigate);
		}
	};

	const handleMarkAllRead = () => {
		markAllReadLocally();
		markAllNotificationsAsRead();
	};

	return (
		<DropdownMenu>
			<DropdownMenuTrigger>
				<div className="relative flex h-10 w-10 items-center justify-center rounded-lg cursor-pointer hover:bg-muted">
					<Bell
						key={`bell-${arrivals}`}
						className={cn("h-5 w-5 origin-top", arrivals > 0 && "animate-ring")}
					/>
					{unreadCount > 0 && (
						<span
							key={`badge-${arrivals}`}
							className={cn(
								"absolute -top-1 -right-1 flex h-5 min-w-5",
								arrivals > 0 && "animate-pop",
							)}
						>
							<Badge
								variant="destructive"
								className="relative h-5 min-w-5 justify-center rounded-full bg-destructive px-1 text-xs text-white"
							>
								{unreadCount > UNREAD_BADGE_MAX
									? `${UNREAD_BADGE_MAX}+`
									: unreadCount}
							</Badge>
						</span>
					)}
				</div>
			</DropdownMenuTrigger>
			<DropdownMenuContent align="end" className="w-80">
				<DropdownMenuGroup>
					<DropdownMenuLabel className="flex items-center justify-between">
						<span>{textGet("notification.bell.title")}</span>
						{unreadCount > 0 && (
							<button
								type="button"
								className="text-xs font-normal text-muted-foreground hover:underline"
								onClick={handleMarkAllRead}
							>
								{textGet("notification.bell.mark_all_read")}
							</button>
						)}
					</DropdownMenuLabel>
				</DropdownMenuGroup>
				<DropdownMenuSeparator />
				{notifications.length === 0 ? (
					<div className="px-2 py-4 text-center text-sm text-muted-foreground">
						{textGet("notification.bell.empty")}
					</div>
				) : (
					notifications.map((notification) => (
						<DropdownMenuItem
							key={notification.ID}
							className="flex flex-col items-start gap-1 whitespace-normal py-2"
							onClick={() => handleSelect(notification)}
						>
							<span className="flex items-start gap-2 font-medium">
								<NotificationIcon
									notification={notification}
									className="mt-0.5"
								/>
								{getNotificationText(notification, { textGet, formatRelative })}
							</span>
							<span className="text-xs text-muted-foreground">
								{formatRelative(notification.CreatedAt)}
							</span>
						</DropdownMenuItem>
					))
				)}
				<DropdownMenuSeparator />
				<DropdownMenuItem
					className="justify-center text-sm text-muted-foreground"
					onClick={() => navigate("/notifications")}
				>
					{textGet("notification.bell.view_all")}
				</DropdownMenuItem>
			</DropdownMenuContent>
		</DropdownMenu>
	);
};

export default NotificationBell;
