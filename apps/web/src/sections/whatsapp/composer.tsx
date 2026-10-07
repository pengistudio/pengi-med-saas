import { useText } from "@pengi/shared";
import { Button, Text, Textarea } from "@pengi/ui";
import { Clock, FileText, Loader2, Lock, Send } from "lucide-react";
import React from "react";
import {
	MAX_REPLY_LENGTH,
	sendWhatsAppReply,
	type WhatsAppMessage,
} from "@/api/whatsapp-service";
import type { ReplyWindow } from "@/lib/whatsapp-window";

interface ComposerProps {
	conversationId: number;
	replyWindow: ReplyWindow;
	/** The reply was accepted (it may still fail later; polling shows it). */
	onSent: (message: WhatsAppMessage) => void;
	/** The reply was rejected; the thread reloads (Meta rejections stay as failed rows). */
	onError: () => void;
	/** Opens the template picker: the only way to write with the window closed. */
	onSendTemplate?: () => void;
}

/** Free-text reply box: Enter sends, Shift+Enter breaks the line. */
export function Composer({
	conversationId,
	replyWindow,
	onSent,
	onError,
	onSendTemplate,
}: ComposerProps) {
	const { textGet } = useText();
	const [body, setBody] = React.useState("");
	const [sending, setSending] = React.useState(false);
	const text = body.trim();
	const canSend = replyWindow.open && !sending && text.length > 0;

	async function send() {
		if (!canSend) return;
		setSending(true);
		const res = await sendWhatsAppReply(conversationId, text);
		setSending(false);
		if (res.success) {
			setBody("");
			onSent(res.data);
		} else {
			onError();
		}
	}

	function onKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
		if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
			e.preventDefault();
			send();
		}
	}

	if (!replyWindow.open) {
		return (
			<div className="flex flex-wrap items-center gap-2 border-t bg-muted/40 p-3 text-sm text-muted-foreground">
				<span className="flex min-w-0 flex-1 basis-60 items-start gap-2">
					<Lock className="mt-0.5 h-4 w-4 shrink-0" />
					<Text uuid="whatsapp.inbox.composer.closed_template" />
				</span>
				{onSendTemplate && (
					<Button size="sm" onClick={onSendTemplate}>
						<FileText />
						<Text uuid="whatsapp.inbox.composer.send_template" />
					</Button>
				)}
			</div>
		);
	}

	return (
		<div className="grid gap-1.5 border-t p-3">
			<div className="flex items-end gap-2">
				<Textarea
					value={body}
					onChange={(e) => setBody(e.target.value)}
					onKeyDown={onKeyDown}
					maxLength={MAX_REPLY_LENGTH}
					rows={2}
					placeholder={textGet("whatsapp.inbox.composer.placeholder")}
					aria-label={textGet("whatsapp.inbox.composer.placeholder")}
					className="max-h-40 min-h-10 resize-none"
				/>
				<Button
					size="icon"
					onClick={send}
					disabled={!canSend}
					aria-label={textGet("whatsapp.inbox.composer.send")}
					title={textGet("whatsapp.inbox.composer.send")}
				>
					{sending ? <Loader2 className="animate-spin" /> : <Send />}
				</Button>
			</div>
			<p className="flex items-center gap-1 text-xs text-muted-foreground">
				<Clock className="h-3 w-3" />
				{replyWindow.hoursLeft > 0 ? (
					<Text
						uuid="whatsapp.inbox.composer.window_hours"
						values={{ count: replyWindow.hoursLeft }}
					/>
				) : (
					<Text
						uuid="whatsapp.inbox.composer.window_minutes"
						values={{ count: replyWindow.minutesLeft }}
					/>
				)}
				{body.length > MAX_REPLY_LENGTH - 200 && (
					<span className="ml-auto tabular-nums">
						{body.length}/{MAX_REPLY_LENGTH}
					</span>
				)}
			</p>
		</div>
	);
}
