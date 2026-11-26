import { useText } from "@pengi/shared";
import {
	Badge,
	Button,
	Card,
	CardContent,
	EmptyState,
	Spinner,
	Tabs,
	TabsList,
	TabsTrigger,
	Text,
} from "@pengi/ui";
import { Check, ChevronLeft, ChevronRight, Trash2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
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
	NotificationAvatar,
	openNotificationLink,
} from "@/lib/notification-level";
import { getNotificationText } from "@/lib/notification-text";
import { cn } from "@/lib/utils";
import { useNotificationStore } from "@/store/notification-store";

const PAGE_LIMIT = 10;

type Filter = "all" | "unread";

const FILTERS: { value: Filter; labelKey: string }[] = [
	{ value: "all", labelKey: "notification.page.filter.all" },
	{ value: "unread", labelKey: "notification.page.filter.unread" },
];

type Group = "today" | "yesterday" | "week" | "older";

const DAY_MS = 86_400_000;

/** Which day bucket a notification falls in, by local calendar day. */
const groupOf = (createdAt: string): Group => {
	const startOfToday = new Date().setHours(0, 0, 0, 0);
	const created = new Date(createdAt).getTime();
	if (created >= startOfToday) return "today";
	if (created >= startOfToday - DAY_MS) return "yesterday";
	if (created >= startOfToday - 6 * DAY_MS) return "week";
	return "older";
};

const NotificationsPage = () => {
	const { textGet, formatRelative, formatDateTime } = useText();
	const navigate = useNavigate();
	const unreadCount = useNotificationStore((s) => s.unreadCount);
	const markReadLocally = useNotificationStore((s) => s.markReadLocally);
	const markAllReadLocally = useNotificationStore((s) => s.markAllReadLocally);
	const removeLocally = useNotificationStore((s) => s.removeLocally);

	// `loading` is only the first fetch. Later page/filter changes keep the
	// current rows on screen (`navigating`) so the list never collapses to a
	// spinner and jumps.
	const [loading, setLoading] = useState(true);
	const [navigating, setNavigating] = useState(false);
	const [items, setItems] = useState<Notification[]>([]);
	const [total, setTotal] = useState(0);
	const [page, setPage] = useState(1);
	const [filter, setFilter] = useState<Filter>("all");
	// Bumped when a navigation's rows arrive, to replay the entrance fade.
	const [viewKey, setViewKey] = useState(0);
	// Rows collapsing out before the next refresh drops them.
	const [leaving, setLeaving] = useState<Set<number>>(new Set());
	const listRef = useRef<HTMLDivElement>(null);
	const latestRequest = useRef(0);

	const totalPages = Math.max(1, Math.ceil(total / PAGE_LIMIT));

	const groups = (["today", "yesterday", "week", "older"] as const)
		.map((group) => ({
			group,
			rows: items.filter((n) => groupOf(n.CreatedAt) === group),
		}))
		.filter((g) => g.rows.length > 0);

	const loadNotifications = useCallback(
		(p: number, f: Filter, animate: boolean) => {
			const request = ++latestRequest.current;
			return getNotifications({
				page: p,
				limit: PAGE_LIMIT,
				unread: f === "unread",
			}).then((res) => {
				// A newer request is in flight: this answer is stale.
				if (request !== latestRequest.current) return;
				if (res.success) {
					setItems(res.data.items);
					setTotal(res.data.total);
					setLeaving(new Set());
					if (animate) setViewKey((k) => k + 1);
				}
				setLoading(false);
				setNavigating(false);
			});
		},
		[],
	);

	// Deleting the last item of the last page leaves it empty — step back
	// (adjust state during render).
	if (page > totalPages) setPage(totalPages);

	// Dim the current rows as soon as the query changes (adjust state during render).
	const [prevQuery, setPrevQuery] = useState({ page, filter });
	if (prevQuery.page !== page || prevQuery.filter !== filter) {
		setPrevQuery({ page, filter });
		setNavigating(true);
	}

	useEffect(() => {
		loadNotifications(page, filter, true);
	}, [page, filter, loadNotifications]);

	// Silent re-sync after an action: no dimming, no replayed entrance.
	const refresh = () => loadNotifications(page, filter, false);

	const goToPage = (next: number) => {
		setPage(next);
		const reduce = window.matchMedia(
			"(prefers-reduced-motion: reduce)",
		).matches;
		listRef.current?.scrollIntoView({
			block: "nearest",
			behavior: reduce ? "auto" : "smooth",
		});
	};

	const markReadInList = (id: number) =>
		setItems((prev) =>
			prev.map((n) =>
				n.ID === id ? { ...n, read_at: new Date().toISOString() } : n,
			),
		);

	const collapse = (id: number) => setLeaving((prev) => new Set(prev).add(id));

	const handleMarkRead = async (notification: Notification) => {
		if (notification.read_at) return;
		markReadLocally(notification.ID);
		// In the unread tab a read row leaves the list; elsewhere it just fades to read.
		if (filter === "unread") collapse(notification.ID);
		else markReadInList(notification.ID);
		await markNotificationAsRead(notification.ID);
		refresh();
	};

	const handleOpen = (notification: Notification) => {
		if (!notification.read_at) {
			markReadLocally(notification.ID);
			markReadInList(notification.ID);
			markNotificationAsRead(notification.ID);
		}
		if (notification.action_url) {
			openNotificationLink(notification.action_url, navigate);
		}
	};

	const handleDelete = async (notification: Notification) => {
		collapse(notification.ID);
		const res = await deleteNotification(notification.ID);
		if (res.success) {
			removeLocally(notification.ID, !notification.read_at);
			refresh();
		} else {
			setLeaving((prev) => {
				const next = new Set(prev);
				next.delete(notification.ID);
				return next;
			});
		}
	};

	const handleMarkAllRead = async () => {
		markAllReadLocally();
		const readAt = new Date().toISOString();
		if (filter === "unread") setLeaving(new Set(items.map((n) => n.ID)));
		else
			setItems((prev) =>
				prev.map((n) => ({ ...n, read_at: n.read_at ?? readAt })),
			);
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
			<Tabs
				value={filter}
				onValueChange={(value) => {
					setFilter(value as Filter);
					setPage(1);
				}}
			>
				<TabsList>
					{FILTERS.map((f) => (
						<TabsTrigger
							key={f.value}
							value={f.value}
							className="gap-1.5 sm:px-3"
						>
							<Text uuid={f.labelKey} />
							{f.value === "unread" && unreadCount > 0 && (
								<Badge>{unreadCount}</Badge>
							)}
						</TabsTrigger>
					))}
				</TabsList>
			</Tabs>

			<Card ref={listRef} className="gap-0 py-0">
				<CardContent className="p-0">
					{loading ? (
						<div className="flex justify-center py-10">
							<Spinner />
						</div>
					) : items.length === 0 ? (
						<EmptyState
							illustration={filter === "unread" ? "all-clear" : "empty"}
							title={textGet(
								filter === "unread"
									? "notification.page.empty_unread"
									: "notification.page.empty",
							)}
						/>
					) : (
						<div
							key={viewKey}
							aria-busy={navigating}
							className={cn(
								"transition-opacity duration-150 motion-safe:animate-in motion-safe:fade-in-0 motion-safe:duration-200",
								navigating && "pointer-events-none opacity-60",
							)}
						>
							{groups.map(({ group, rows }) => (
								<section key={group}>
									<h2 className="border-b bg-muted/40 px-4 py-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
										{textGet(`notification.page.group.${group}`)}
									</h2>
									<ul>
										{rows.map((notification) => (
											<li
												key={notification.ID}
												// Collapses (rows 1fr -> 0fr) instead of vanishing, so the rows
												// below slide up rather than jump.
												className={cn(
													"grid border-b transition-[grid-template-rows,opacity] duration-200 ease-out last:border-b-0 motion-reduce:transition-none",
													leaving.has(notification.ID)
														? "pointer-events-none grid-rows-[0fr] border-b-0 opacity-0"
														: "grid-rows-[1fr]",
												)}
											>
												<div
													className={cn(
														"group relative flex min-h-0 items-center gap-3 overflow-hidden px-4 py-3 transition-colors hover:bg-muted/40",
														!notification.read_at && "bg-primary/[0.03]",
													)}
												>
													<NotificationAvatar notification={notification} />
													<button
														type="button"
														className="flex min-w-0 flex-1 flex-col items-start gap-0.5 text-left outline-none focus-visible:underline"
														onClick={() => handleOpen(notification)}
													>
														<span
															className={cn(
																"text-sm transition-colors duration-200",
																notification.read_at
																	? "text-muted-foreground"
																	: "font-medium text-foreground",
															)}
														>
															{getNotificationText(notification, {
																textGet,
																formatRelative,
															})}
														</span>
														<time
															className="text-xs text-muted-foreground"
															dateTime={notification.CreatedAt}
															title={formatDateTime(notification.CreatedAt)}
														>
															{formatRelative(notification.CreatedAt)}
														</time>
													</button>
													{/* Dot and actions share one slot: the dot gives way to the buttons. */}
													<div className="relative flex h-9 w-[4.75rem] shrink-0 items-center justify-end">
														<span
															aria-hidden
															className={cn(
																"absolute right-3 h-2 w-2 rounded-full bg-primary transition duration-200 group-focus-within:opacity-0 group-hover:opacity-0 [@media(hover:none)]:hidden",
																notification.read_at && "scale-0",
															)}
														/>
														<div className="flex gap-1 opacity-0 transition-opacity duration-150 group-focus-within:opacity-100 group-hover:opacity-100 [@media(hover:none)]:opacity-100">
															{!notification.read_at && (
																<Button
																	variant="ghost"
																	size="icon"
																	className="transition-transform active:scale-90"
																	aria-label={textGet(
																		"notification.page.mark_read",
																	)}
																	title={textGet("notification.page.mark_read")}
																	onClick={() => handleMarkRead(notification)}
																>
																	<Check className="h-4 w-4" />
																</Button>
															)}
															<Button
																variant="ghost"
																size="icon"
																className="transition-transform active:scale-90"
																aria-label={textGet("notification.page.delete")}
																title={textGet("notification.page.delete")}
																onClick={() => handleDelete(notification)}
															>
																<Trash2 className="h-4 w-4" />
															</Button>
														</div>
													</div>
												</div>
											</li>
										))}
									</ul>
								</section>
							))}
						</div>
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
						disabled={page <= 1 || navigating}
						onClick={() => goToPage(page - 1)}
					>
						<ChevronLeft className="h-4 w-4" />
					</Button>
					<Button
						variant="outline"
						size="icon"
						aria-label={textGet("table.pagination.next_page")}
						disabled={page >= totalPages || navigating}
						onClick={() => goToPage(page + 1)}
					>
						<ChevronRight className="h-4 w-4" />
					</Button>
				</div>
			)}
		</div>
	);
};

export default NotificationsPage;
