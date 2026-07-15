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
import { useNavigate } from "react-router";
import {
	markAllNotificationsAsRead,
	markNotificationAsRead,
	type Notification,
} from "@/api/notification-service";
import { useText } from "@/hooks/use-text";
import {
	formatElapsedDuration,
	formatRelativeTime,
	renderNotificationText,
} from "@/lib/notification-text";
import { useMessageStore } from "@/store/message-store";
import { useNotificationStore } from "@/store/notification-store";

const UNREAD_BADGE_MAX = 9;

const NotificationBell = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const lang = useMessageStore((s) => s.lang);
	const notifications = useNotificationStore((s) => s.notifications);
	const unreadCount = useNotificationStore((s) => s.unreadCount);
	const markReadLocally = useNotificationStore((s) => s.markReadLocally);
	const markAllReadLocally = useNotificationStore((s) => s.markAllReadLocally);

	const handleSelect = (notification: Notification) => {
		if (!notification.read_at) {
			markReadLocally(notification.ID);
			markNotificationAsRead(notification.ID);
		}
		if (notification.action_url) {
			navigate(notification.action_url);
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
					<Bell className="h-5 w-5" />
					{unreadCount > 0 && (
						<span className="absolute -top-1 -right-1 flex h-5 min-w-5">
							<span className="absolute inset-0.5 animate-ping rounded-full bg-destructive opacity-60" />
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
					notifications.map((notification) => {
						const params = notification.params.draft_updated_at
							? {
									...notification.params,
									elapsed: formatElapsedDuration(
										notification.params.draft_updated_at,
										lang,
									),
								}
							: notification.params;

						return (
							<DropdownMenuItem
								key={notification.ID}
								className="flex flex-col items-start gap-1 whitespace-normal py-2"
								onClick={() => handleSelect(notification)}
							>
								<span
									className={
										notification.read_at
											? "text-muted-foreground"
											: "font-medium"
									}
								>
									{renderNotificationText(
										textGet(notification.message_key),
										params,
									)}
								</span>
								<span className="text-xs text-muted-foreground">
									{formatRelativeTime(notification.CreatedAt, lang)}
								</span>
							</DropdownMenuItem>
						);
					})
				)}
			</DropdownMenuContent>
		</DropdownMenu>
	);
};

export default NotificationBell;
