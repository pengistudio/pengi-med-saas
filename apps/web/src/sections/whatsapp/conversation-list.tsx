import { useText } from "@pengi/shared";
import { Button, cn, Input, Skeleton, Text } from "@pengi/ui";
import { MessagesSquare, Search } from "lucide-react";
import React from "react";
import {
	getWhatsAppConversations,
	type WhatsAppConversation,
} from "@/api/whatsapp-service";
import { usePolling } from "@/hooks/use-polling";
import { conversationTitle } from "@/lib/whatsapp-thread";

const LIST_POLL_MS = 15_000;
const PAGE_SIZE = 30;
/** The backend serves at most 100 rows per page. */
const MAX_LIMIT = 100;
const SEARCH_DEBOUNCE_MS = 300;
const MEDIA_TYPES = new Set(["image", "audio", "document", "other"]);

interface ConversationListProps {
	selectedId: number | null;
	onSelect: (id: number) => void;
	/** Changing it reloads the list (after reading or replying). */
	refreshKey: number;
	className?: string;
}

/** Inbox list: search by name or number, "unread" filter, newest first. */
export function ConversationList({
	selectedId,
	onSelect,
	refreshKey,
	className,
}: ConversationListProps) {
	const { textGet, formatRelative } = useText();
	const [search, setSearch] = React.useState("");
	const [query, setQuery] = React.useState("");
	const [unreadOnly, setUnreadOnly] = React.useState(false);
	const [limit, setLimit] = React.useState(PAGE_SIZE);

	React.useEffect(() => {
		const id = setTimeout(() => setQuery(search.trim()), SEARCH_DEBOUNCE_MS);
		return () => clearTimeout(id);
	}, [search]);

	const paramsKey = `${query}|${unreadOnly}|${limit}`;
	// The rows with the params they answer: loading while they don't match.
	const [data, setData] = React.useState<{
		key: string;
		items: WhatsAppConversation[];
		total: number;
	}>({ key: "", items: [], total: 0 });
	const loading = data.key !== paramsKey;
	const latestKey = React.useRef(paramsKey);
	React.useEffect(() => {
		latestKey.current = paramsKey;
	}, [paramsKey]);

	React.useEffect(() => {
		let cancelled = false;
		getWhatsAppConversations(
			{ search: query, unread: unreadOnly, limit, page: 1 },
			{ notifyError: true },
		).then((res) => {
			if (cancelled) return;
			setData(
				res.success
					? {
							key: paramsKey,
							items: res.data.items ?? [],
							total: res.data.total,
						}
					: { key: paramsKey, items: [], total: 0 },
			);
		});
		return () => {
			cancelled = true;
		};
	}, [query, unreadOnly, limit, paramsKey, refreshKey]);

	// Polls fail silently and drop answers for params the user already changed.
	const poll = React.useCallback(async () => {
		const key = paramsKey;
		const res = await getWhatsAppConversations({
			search: query,
			unread: unreadOnly,
			limit,
			page: 1,
		});
		if (res.success && key === latestKey.current) {
			setData({ key, items: res.data.items ?? [], total: res.data.total });
		}
	}, [query, unreadOnly, limit, paramsKey]);
	usePolling(poll, LIST_POLL_MS);

	function preview(c: WhatsAppConversation) {
		const text =
			c.last_message_preview ||
			(MEDIA_TYPES.has(c.last_content_type)
				? textGet(`whatsapp.inbox.content.${c.last_content_type}`)
				: "");
		return c.last_direction === "outbound" && text
			? textGet("whatsapp.inbox.preview.you", { text })
			: text;
	}

	return (
		<div className={cn("flex min-h-0 flex-col", className)}>
			<div className="grid gap-2 border-b p-3">
				<div className="relative">
					<Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						value={search}
						onChange={(e) => setSearch(e.target.value)}
						placeholder={textGet("whatsapp.inbox.search")}
						aria-label={textGet("whatsapp.inbox.search")}
						className="pl-9"
					/>
				</div>
				<div className="flex gap-2">
					<Button
						size="sm"
						variant={unreadOnly ? "outline" : "secondary"}
						onClick={() => setUnreadOnly(false)}
					>
						<Text uuid="whatsapp.inbox.filter.all" />
					</Button>
					<Button
						size="sm"
						variant={unreadOnly ? "secondary" : "outline"}
						onClick={() => setUnreadOnly(true)}
					>
						<Text uuid="whatsapp.inbox.filter.unread" />
					</Button>
				</div>
			</div>
			<div className="min-h-0 flex-1 overflow-y-auto">
				{loading && data.items.length === 0 ? (
					<div className="grid gap-3 p-3">
						{[0, 1, 2, 3].map((i) => (
							<Skeleton key={i} className="h-12 w-full" />
						))}
					</div>
				) : data.items.length === 0 ? (
					<div className="flex flex-col items-center gap-2 p-8 text-center text-sm text-muted-foreground">
						<MessagesSquare className="h-8 w-8" />
						<Text
							uuid={
								query || unreadOnly
									? "whatsapp.inbox.list.no_results"
									: "whatsapp.inbox.list.empty"
							}
						/>
					</div>
				) : (
					<ul>
						{data.items.map((c) => {
							const title = conversationTitle(c);
							return (
								<li key={c.id}>
									<button
										type="button"
										onClick={() => onSelect(c.id)}
										aria-current={c.id === selectedId ? "true" : undefined}
										className={cn(
											"flex w-full items-center gap-3 border-b px-3 py-2.5 text-left hover:bg-accent/60",
											c.id === selectedId && "bg-accent",
										)}
									>
										<span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-emerald-500/15 text-sm font-semibold text-emerald-700 dark:text-emerald-400">
											{title.replace("+", "").charAt(0).toUpperCase()}
										</span>
										<span className="grid min-w-0 flex-1">
											<span className="flex items-baseline justify-between gap-2">
												<span
													className={cn(
														"truncate text-sm",
														c.unread_count > 0
															? "font-semibold"
															: "font-medium",
													)}
												>
													{title}
												</span>
												<span className="shrink-0 text-[11px] text-muted-foreground">
													{formatRelative(c.last_message_at ?? c.created_at)}
												</span>
											</span>
											<span className="flex items-center justify-between gap-2">
												<span className="truncate text-xs text-muted-foreground">
													{preview(c)}
												</span>
												{c.unread_count > 0 && (
													<span className="flex h-5 min-w-5 shrink-0 items-center justify-center rounded-full bg-emerald-600 px-1.5 text-[11px] font-semibold text-white">
														{c.unread_count}
													</span>
												)}
											</span>
										</span>
									</button>
								</li>
							);
						})}
					</ul>
				)}
				{data.total > data.items.length && limit >= MAX_LIMIT && (
					<p className="p-3 text-center text-xs text-muted-foreground">
						<Text uuid="whatsapp.inbox.list.use_search" />
					</p>
				)}
				{data.total > data.items.length && limit < MAX_LIMIT && (
					<div className="p-3">
						<Button
							variant="ghost"
							size="sm"
							className="w-full"
							disabled={loading}
							onClick={() =>
								setLimit((l) => Math.min(l + PAGE_SIZE, MAX_LIMIT))
							}
						>
							<Text uuid="whatsapp.inbox.list.more" />
						</Button>
					</div>
				)}
			</div>
		</div>
	);
}
