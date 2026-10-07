import { useText } from "@pengi/shared";
import { cn, Text } from "@pengi/ui";
import {
	AlertCircle,
	BellRing,
	Bot,
	Check,
	CheckCheck,
	Clock,
	FileText,
	FlaskConical,
	X,
} from "lucide-react";
import {
	messageErrorKey,
	type WhatsAppMessage,
	type WhatsAppMessageStatus,
} from "@/api/whatsapp-service";

/** Content we can't show yet (media isn't downloaded in v1). */
const MEDIA_TYPES = new Set(["image", "audio", "document", "other"]);

function StatusTick({ status }: { status: WhatsAppMessageStatus }) {
	const { textGet } = useText();
	const label = textGet(`settings.whatsapp.log.status.${status}`);
	const props = { className: "h-3.5 w-3.5", "aria-label": label };
	switch (status) {
		case "queued":
		case "sending":
			return (
				<span title={label}>
					<Clock {...props} />
				</span>
			);
		case "sent":
			return (
				<span title={label}>
					<Check {...props} />
				</span>
			);
		case "delivered":
			return (
				<span title={label}>
					<CheckCheck {...props} />
				</span>
			);
		case "read":
		case "replied":
			return (
				<span title={label} className="text-sky-600 dark:text-sky-400">
					<CheckCheck {...props} />
				</span>
			);
		case "failed":
		case "skipped":
			return (
				<span title={label} className="text-destructive">
					<AlertCircle {...props} />
				</span>
			);
		default:
			return null;
	}
}

/** One bubble of a conversation: inbound on the left, outbound on the right. */
export function MessageBubble({ message }: { message: WhatsAppMessage }) {
	const { formatTime } = useText();
	const inbound = message.direction === "inbound";
	const errorKey =
		message.status === "failed" || message.status === "skipped"
			? messageErrorKey(message.error_code)
			: null;
	const isMedia = MEDIA_TYPES.has(message.content_type);

	return (
		<div className={cn("flex", inbound ? "justify-start" : "justify-end")}>
			<div
				className={cn(
					"grid max-w-[85%] gap-1 rounded-2xl px-3 py-2 text-sm shadow-xs sm:max-w-[70%]",
					inbound
						? "rounded-bl-sm bg-muted"
						: "rounded-br-sm bg-emerald-100 text-emerald-950 dark:bg-emerald-900/50 dark:text-emerald-50",
					message.status === "failed" && "ring-1 ring-destructive/40",
				)}
			>
				{message.kind === "reminder" && (
					<span className="inline-flex w-fit items-center gap-1 rounded-full bg-background/70 px-2 py-0.5 text-xs font-medium">
						<BellRing className="h-3 w-3" />
						<Text
							uuid="whatsapp.inbox.reminder_label"
							values={{ count: message.offset_hours }}
						/>
					</span>
				)}
				{message.kind === "test" && (
					<span className="inline-flex w-fit items-center gap-1 rounded-full bg-background/70 px-2 py-0.5 text-xs font-medium">
						<FlaskConical className="h-3 w-3" />
						<Text uuid="settings.whatsapp.log.kind.test" />
					</span>
				)}
				{message.kind === "template" && (
					<span className="inline-flex w-fit items-center gap-1 rounded-full bg-background/70 px-2 py-0.5 text-xs font-medium">
						<FileText className="h-3 w-3" />
						<Text uuid={`whatsapp.template.name.${message.template}`} />
					</span>
				)}
				{message.kind === "system" && (
					<span className="inline-flex w-fit items-center gap-1 rounded-full bg-background/70 px-2 py-0.5 text-xs font-medium">
						<Bot className="h-3 w-3" />
						<Text uuid="whatsapp.inbox.system_label" />
					</span>
				)}
				{inbound && message.reply && (
					<span
						className={cn(
							"inline-flex w-fit items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
							message.reply === "confirm"
								? "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400"
								: "bg-orange-500/15 text-orange-700 dark:text-orange-400",
						)}
					>
						{message.reply === "confirm" ? (
							<Check className="h-3 w-3" />
						) : (
							<X className="h-3 w-3" />
						)}
						<Text uuid={`whatsapp.inbox.reply.${message.reply}`} />
					</span>
				)}
				{isMedia && (
					<span className="text-xs italic text-muted-foreground">
						<Text uuid={`whatsapp.inbox.content.${message.content_type}`} />
					</span>
				)}
				{message.body ? (
					<p className="whitespace-pre-wrap break-words">{message.body}</p>
				) : (
					message.kind === "reminder" && (
						<p className="text-muted-foreground">
							<Text
								uuid="whatsapp.inbox.reminder_template"
								values={{ template: message.template }}
							/>
						</p>
					)
				)}
				{errorKey && (
					<p
						className="text-xs text-destructive"
						title={message.error_detail || undefined}
					>
						<Text uuid={errorKey} />
					</p>
				)}
				<span className="flex items-center justify-end gap-1 text-[11px] text-muted-foreground">
					{formatTime(message.created_at)}
					{!inbound && <StatusTick status={message.status} />}
				</span>
			</div>
		</div>
	);
}
