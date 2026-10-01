import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	Spinner,
	Text,
	ToggleGroup,
	ToggleGroupItem,
} from "@pengi/ui";
import { Check, ChevronLeft, ChevronRight, Trash2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import {
	deleteNotification,
	deleteReadNotifications,
	getNotifications,
	markAllNotificationsAsRead,
	markNotificationAsRead,
	type Notification,
} from "@/api/notification-service";
import { PageHeader } from "@/components/custom/page-header";
import {
	NotificationLevelIcon,
	openNotificationLink,
} from "@/lib/notification-level";
import { getNotificationText } from "@/lib/notification-text";
import { cn } from "@/lib/utils";
import { useNotificationStore } from "@/store/notification-store";

const PAGE_LIMIT = 20;

type Filter = "all" | "unread";

const FILTERS: { value: Filter; labelKey: string }[] = [
	{ value: "all", labelKey: "notification.page.filter.all" },
	{ value: "unread", labelKey: "notification.page.filter.unread" },
];

const NotificationsPage = () => {
	const { textGet, formatRelative } = useText();
	const navigate = useNavigate();
	const unreadCount = useNotificationStore((s) => s.unreadCount);
	const markReadLocally = useNotificationStore((s) => s.markReadLocally);
	const markAllReadLocally = useNotificationStore((s) => s.markAllReadLocally);
	const removeLocally = useNotificationStore((s) => s.removeLocally);

	const [loading, setLoading] = useState(true);
	const [items, setItems] = useState<Notification[]>([]);
	const [total, setTotal] = useState(0);
	const [page, setPage] = useState(1);
	const [filter, setFilter] = useState<Filter>("all");

	const totalPages = Math.max(1, Math.ceil(total / PAGE_LIMIT));

	// Requests the page without flagging loading; callers set it first.
	const loadNotifications = useCallback(
		(p: number, f: Filter) =>
			getNotifications({
				page: p,
				limit: PAGE_LIMIT,
				unread: f === "unread",
			}).then((res) => {
				if (res.success) {
					setItems(res.data.items);
					setTotal(res.data.total);
				}
				setLoading(false);
			}),
		[],
	);

	// Deleting the last item of the last page leaves it empty — step back
	// (adjust state during render).
	if (page > totalPages) setPage(totalPages);

	// Show the spinner as soon as the query changes (adjust state during render).
	const [prevQuery, setPrevQuery] = useState({ page, filter });
	if (prevQuery.page !== page || prevQuery.filter !== filter) {
		setPrevQuery({ page, filter });
		setLoading(true);
	}

	useEffect(() => {
		loadNotifications(page, filter);
	}, [page, filter, loadNotifications]);

	const refresh = () => {
		setLoading(true);
		return loadNotifications(page, filter);
	};

	const handleMarkRead = async (notification: Notification) => {
		if (notification.read_at) return;
		markReadLocally(notification.ID);
		await markNotificationAsRead(notification.ID);
		refresh();
	};

	const handleOpen = (notification: Notification) => {
		if (!notification.read_at) {
			markReadLocally(notification.ID);
			markNotificationAsRead(notification.ID);
		}
		if (notification.action_url) {
			openNotificationLink(notification.action_url, navigate);
		} else {
			refresh();
		}
	};

	const handleDelete = async (notification: Notification) => {
		const res = await deleteNotification(notification.ID);
		if (res.success) {
			removeLocally(notification.ID, !notification.read_at);
			refresh();
		}
	};

	const handleMarkAllRead = async () => {
		markAllReadLocally();
		await markAllNotificationsAsRead();
		refresh();
	};

	const handleDeleteRead = async () => {
		const res = await deleteReadNotifications();
		if (res.success) refresh();
	};

	return (
		<div className="space-y-6">
			<PageHeader
				title={textGet("notification.page.title")}
				description={textGet("notification.page.description")}
				actions={
					<>
						<Button
							variant="outline"
							disabled={unreadCount === 0}
							onClick={handleMarkAllRead}
						>
							<Check className="mr-2 h-4 w-4" />
							<Text uuid="notification.page.mark_all_read" />
						</Button>
						<Button variant="outline" onClick={handleDeleteRead}>
							<Trash2 className="mr-2 h-4 w-4" />
							<Text uuid="notification.page.delete_read" />
						</Button>
					</>
				}
			/>
			<ToggleGroup
				value={[filter]}
				onValueChange={(value) => {
					setFilter((value[0] as Filter | undefined) ?? "all");
					setPage(1);
				}}
			>
				{FILTERS.map((f) => (
					<ToggleGroupItem key={f.value} value={f.value}>
						<Text uuid={f.labelKey} />
					</ToggleGroupItem>
				))}
			</ToggleGroup>

			<Card>
				<CardContent className="p-0">
					{loading ? (
						<div className="flex justify-center py-10">
							<Spinner />
						</div>
					) : items.length === 0 ? (
						<p className="py-10 text-center text-sm text-muted-foreground">
							{textGet(
								filter === "unread"
									? "notification.page.empty_unread"
									: "notification.page.empty",
							)}
						</p>
					) : (
						<ul className="divide-y">
							{items.map((notification) => (
								<li
									key={notification.ID}
									className="flex items-start gap-3 px-4 py-3"
								>
									<span
										className={cn(
											"mt-2 h-2 w-2 shrink-0 rounded-full",
											notification.read_at ? "bg-transparent" : "bg-primary",
										)}
									/>
									<button
										type="button"
										className="flex flex-1 flex-col items-start gap-1 text-left"
										onClick={() => handleOpen(notification)}
									>
										<span
											className={cn(
												"flex items-start gap-2 text-sm",
												notification.read_at
													? "text-muted-foreground"
													: "font-medium",
											)}
										>
											<NotificationLevelIcon
												level={notification.level}
												className="mt-0.5"
											/>
											{getNotificationText(notification, {
												textGet,
												formatRelative,
											})}
										</span>
										<span className="text-xs text-muted-foreground">
											{formatRelative(notification.CreatedAt)}
										</span>
									</button>
									<div className="flex shrink-0 gap-1">
										{!notification.read_at && (
											<Button
												variant="ghost"
												size="icon"
												aria-label={textGet("notification.page.mark_read")}
												title={textGet("notification.page.mark_read")}
												onClick={() => handleMarkRead(notification)}
											>
												<Check className="h-4 w-4" />
											</Button>
										)}
										<Button
											variant="ghost"
											size="icon"
											aria-label={textGet("notification.page.delete")}
											title={textGet("notification.page.delete")}
											onClick={() => handleDelete(notification)}
										>
											<Trash2 className="h-4 w-4" />
										</Button>
									</div>
								</li>
							))}
						</ul>
					)}
				</CardContent>
			</Card>

			{totalPages > 1 && (
				<div className="flex items-center justify-end gap-2 text-sm">
					<span className="text-muted-foreground">
						{textGet("table.pagination.page")}
						{page}
						{textGet("table.pagination.of")}
						{totalPages}
					</span>
					<Button
						variant="outline"
						size="icon"
						aria-label={textGet("table.pagination.previous_page")}
						disabled={page <= 1}
						onClick={() => setPage((p) => p - 1)}
					>
						<ChevronLeft className="h-4 w-4" />
					</Button>
					<Button
						variant="outline"
						size="icon"
						aria-label={textGet("table.pagination.next_page")}
						disabled={page >= totalPages}
						onClick={() => setPage((p) => p + 1)}
					>
						<ChevronRight className="h-4 w-4" />
					</Button>
				</div>
			)}
		</div>
	);
};

export default NotificationsPage;
