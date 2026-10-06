import { parseDateOnly, useText } from "@pengi/shared";
import { Button, cn, Skeleton, Text } from "@pengi/ui";
import {
	ArrowLeft,
	BellOff,
	CalendarClock,
	Loader2,
	UserPlus,
} from "lucide-react";
import React from "react";
import { Link } from "react-router";
import {
	getWhatsAppConversation,
	getWhatsAppThread,
	markWhatsAppConversationRead,
	type WhatsAppConversationDetail,
	type WhatsAppMessage,
} from "@/api/whatsapp-service";
import { usePolling } from "@/hooks/use-polling";
import {
	conversationTitle,
	displayPhone,
	mergeMessages,
} from "@/lib/whatsapp-thread";
import { replyWindow } from "@/lib/whatsapp-window";
import { refreshWhatsAppUnread } from "@/store/whatsapp-store";
import { Composer } from "./composer";
import { LinkPatientDialog } from "./link-patient-dialog";
import { MessageBubble } from "./message-bubble";
import { TemplateMessageDialog } from "./template-message-dialog";

const THREAD_POLL_MS = 5_000;
/** The reply window countdown moves once a minute. */
const CLOCK_TICK_MS = 60_000;
/** Within this distance of the bottom, new messages keep the view pinned. */
const STICK_TO_BOTTOM_PX = 120;

interface ConversationThreadProps {
	conversationId: number;
	/** Back to the list (phones). */
	onBack: () => void;
	/** The conversation changed (read, replied, linked): the list reloads. */
	onChanged: () => void;
	className?: string;
}

/**
 * One conversation: header with the patient, the messages and the composer.
 * Mount it with `key={conversationId}` so switching conversations resets it.
 */
export function ConversationThread({
	conversationId,
	onBack,
	onChanged,
	className,
}: ConversationThreadProps) {
	const { textGet, formatDate } = useText();
	const [detail, setDetail] = React.useState<WhatsAppConversationDetail | null>(
		null,
	);
	const [messages, setMessages] = React.useState<WhatsAppMessage[]>([]);
	const [hasMore, setHasMore] = React.useState(false);
	const [loaded, setLoaded] = React.useState(false);
	const [loadingOlder, setLoadingOlder] = React.useState(false);
	const [linkOpen, setLinkOpen] = React.useState(false);
	const [templateOpen, setTemplateOpen] = React.useState(false);
	const [now, setNow] = React.useState(() => new Date());

	const scrollRef = React.useRef<HTMLDivElement>(null);
	/** How the next render of the messages moves the scroll. */
	const scrollMode = React.useRef<
		{ kind: "bottom" } | { kind: "keep" } | { kind: "older"; height: number }
	>({ kind: "bottom" });
	/** Newest message id seen, to spot new ones between polls. */
	const newestId = React.useRef(0);
	// Latest callback without re-running the load and poll effects.
	const onChangedRef = React.useRef(onChanged);
	React.useEffect(() => {
		onChangedRef.current = onChanged;
	}, [onChanged]);

	const markRead = React.useCallback(async () => {
		const res = await markWhatsAppConversationRead(conversationId);
		if (res.success) {
			setDetail((d) => (d ? { ...d, unread_count: 0 } : d));
			refreshWhatsAppUnread();
			onChangedRef.current();
		}
	}, [conversationId]);

	const nearBottom = () => {
		const el = scrollRef.current;
		return (
			!el ||
			el.scrollHeight - el.scrollTop - el.clientHeight < STICK_TO_BOTTOM_PX
		);
	};

	// First load: header + newest page; mark read when it has unread messages.
	React.useEffect(() => {
		let cancelled = false;
		Promise.all([
			getWhatsAppConversation(conversationId, { notifyError: true }),
			getWhatsAppThread(conversationId, {}, { notifyError: true }),
		]).then(([d, t]) => {
			if (cancelled) return;
			if (d.success) setDetail(d.data);
			if (t.success) {
				const items = t.data.items ?? [];
				scrollMode.current = { kind: "bottom" };
				setMessages(items);
				setHasMore(t.data.has_more);
				newestId.current = items.at(-1)?.id ?? 0;
			}
			setLoaded(true);
			if (d.success && d.data.unread_count > 0) markRead();
		});
		return () => {
			cancelled = true;
		};
	}, [conversationId, markRead]);

	// Polls the newest page; a new message refreshes the header (the reply
	// window reopens on inbound) and an inbound one is marked read.
	const poll = React.useCallback(async () => {
		const res = await getWhatsAppThread(conversationId);
		if (!res.success) return;
		const items = res.data.items ?? [];
		const previousNewest = newestId.current;
		const fresh = items.filter((m) => m.id > previousNewest);
		scrollMode.current = { kind: nearBottom() ? "bottom" : "keep" };
		setMessages((prev) => mergeMessages(prev, items));
		if (fresh.length === 0) return;
		newestId.current = Math.max(previousNewest, ...fresh.map((m) => m.id));
		const d = await getWhatsAppConversation(conversationId);
		if (d.success) setDetail(d.data);
		if (fresh.some((m) => m.direction === "inbound")) markRead();
		else onChangedRef.current();
	}, [conversationId, markRead]);
	usePolling(poll, THREAD_POLL_MS, loaded);

	React.useEffect(() => {
		const id = setInterval(() => setNow(new Date()), CLOCK_TICK_MS);
		return () => clearInterval(id);
	}, []);

	React.useLayoutEffect(() => {
		const el = scrollRef.current;
		if (!el || messages.length === 0) return;
		const mode = scrollMode.current;
		if (mode.kind === "bottom") el.scrollTop = el.scrollHeight;
		else if (mode.kind === "older")
			el.scrollTop = el.scrollHeight - mode.height;
		scrollMode.current = { kind: "keep" };
	}, [messages]);

	async function loadOlder() {
		const oldest = messages[0];
		if (!oldest) return;
		setLoadingOlder(true);
		const res = await getWhatsAppThread(
			conversationId,
			{ before: oldest.id },
			{ notifyError: true },
		);
		setLoadingOlder(false);
		if (!res.success) return;
		const el = scrollRef.current;
		scrollMode.current = {
			kind: "older",
			height: el ? el.scrollHeight - el.scrollTop : 0,
		};
		setMessages((prev) => mergeMessages(prev, res.data.items ?? []));
		setHasMore(res.data.has_more);
	}

	function onSent(message: WhatsAppMessage) {
		scrollMode.current = { kind: "bottom" };
		newestId.current = Math.max(newestId.current, message.id);
		setMessages((prev) => mergeMessages(prev, [message]));
		onChanged();
	}

	const replyWin = replyWindow(detail?.window_expires_at, now);
	const patient = detail?.patient ?? null;
	const next = detail?.next_appointment ?? null;

	return (
		<div className={cn("flex min-h-0 flex-col", className)}>
			<div className="flex items-start gap-2 border-b p-3">
				<Button
					variant="ghost"
					size="icon"
					className="md:hidden"
					onClick={onBack}
					aria-label={textGet("whatsapp.inbox.back")}
				>
					<ArrowLeft />
				</Button>
				{detail ? (
					<div className="grid min-w-0 flex-1 gap-1">
						<div className="flex flex-wrap items-center gap-x-2">
							{patient ? (
								<Link
									to={`/clinical/medical-records/${patient.id}`}
									className="truncate font-semibold hover:underline"
								>
									{conversationTitle(detail)}
								</Link>
							) : (
								<span className="truncate font-semibold">
									{conversationTitle(detail)}
								</span>
							)}
							{patient && (
								<span className="text-xs text-muted-foreground tabular-nums">
									{displayPhone(detail.phone)}
								</span>
							)}
						</div>
						{next && (
							<span className="flex items-center gap-1 text-xs text-muted-foreground">
								<CalendarClock className="h-3.5 w-3.5" />
								<Text
									uuid="whatsapp.inbox.next_appointment"
									values={{
										date: formatDate(parseDateOnly(next.date), "medium"),
										time: next.start_time,
									}}
								/>
							</span>
						)}
						{patient && !patient.whatsapp_opt_in && (
							<span className="flex items-center gap-1 text-xs text-amber-700 dark:text-amber-400">
								<BellOff className="h-3.5 w-3.5" />
								<Text uuid="whatsapp.inbox.no_opt_in" />
							</span>
						)}
					</div>
				) : (
					<Skeleton className="h-10 flex-1" />
				)}
				{detail && !patient && (
					<Button variant="outline" size="sm" onClick={() => setLinkOpen(true)}>
						<UserPlus />
						<Text uuid="whatsapp.inbox.link.open" />
					</Button>
				)}
			</div>

			<div
				ref={scrollRef}
				className="min-h-0 flex-1 space-y-2 overflow-y-auto bg-muted/20 p-3"
			>
				{hasMore && (
					<div className="flex justify-center">
						<Button
							variant="ghost"
							size="sm"
							disabled={loadingOlder}
							onClick={loadOlder}
						>
							{loadingOlder && <Loader2 className="animate-spin" />}
							<Text uuid="whatsapp.inbox.thread.older" />
						</Button>
					</div>
				)}
				{!loaded ? (
					<div className="grid gap-2">
						<Skeleton className="h-10 w-2/3" />
						<Skeleton className="ml-auto h-10 w-1/2" />
						<Skeleton className="h-10 w-1/2" />
					</div>
				) : messages.length === 0 ? (
					<p className="py-8 text-center text-sm text-muted-foreground">
						<Text uuid="whatsapp.inbox.thread.empty" />
					</p>
				) : (
					messages.map((m, i) => {
						const day = formatDate(m.created_at, "long");
						const showDay =
							i === 0 || formatDate(messages[i - 1].created_at, "long") !== day;
						return (
							<React.Fragment key={m.id}>
								{showDay && (
									<div className="flex justify-center py-1">
										<span className="rounded-full bg-background px-2.5 py-0.5 text-[11px] text-muted-foreground shadow-xs">
											{day}
										</span>
									</div>
								)}
								<MessageBubble message={m} />
							</React.Fragment>
						);
					})
				)}
			</div>

			{detail && (
				<Composer
					conversationId={conversationId}
					replyWindow={replyWin}
					onSent={onSent}
					onError={poll}
					onSendTemplate={() => setTemplateOpen(true)}
				/>
			)}

			{detail && (
				<TemplateMessageDialog
					open={templateOpen}
					onOpenChange={setTemplateOpen}
					conversationId={conversationId}
					patient={
						patient
							? {
									id: patient.id,
									name: `${patient.first_name} ${patient.last_name}`.trim(),
									phone: patient.phone || displayPhone(detail.phone),
									whatsapp_opt_in: patient.whatsapp_opt_in,
								}
							: null
					}
					onSent={({ message, conversation }) => {
						setDetail((d) => (d ? { ...d, ...conversation } : d));
						onSent(message);
					}}
				/>
			)}

			{detail && (
				<LinkPatientDialog
					open={linkOpen}
					onOpenChange={setLinkOpen}
					conversationId={conversationId}
					phone={displayPhone(detail.phone)}
					onLinked={(d) => {
						setDetail(d);
						onChanged();
					}}
				/>
			)}
		</div>
	);
}
