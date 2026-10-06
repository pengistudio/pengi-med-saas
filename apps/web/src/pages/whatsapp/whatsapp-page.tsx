import { useText } from "@pengi/shared";
import {
	Button,
	cn,
	Tabs,
	TabsContent,
	TabsList,
	TabsTrigger,
	Text,
} from "@pengi/ui";
import {
	MessageSquarePlus,
	MessagesSquare,
	Volume2,
	VolumeX,
} from "lucide-react";
import React from "react";
import { useSearchParams } from "react-router";
import { PageHeader } from "@/components/custom/page-header";
import {
	isNotificationSoundMuted,
	setNotificationSoundMuted,
} from "@/lib/notification-sound";
import { ConversationList } from "@/sections/whatsapp/conversation-list";
import { ConversationThread } from "@/sections/whatsapp/conversation-thread";
import { SentMessages } from "@/sections/whatsapp/sent-messages";
import { TemplateMessageDialog } from "@/sections/whatsapp/template-message-dialog";

type InboxTab = "conversations" | "sent";

/**
 * WhatsApp inbox: conversations (list + thread; on phones one at a time) and
 * the log of sent messages. `?c=<id>` opens a conversation, `?tab=sent` the log.
 * "Nuevo mensaje" writes first to a patient with a template.
 */
export default function WhatsAppPage() {
	const { textGet } = useText();
	const [params, setParams] = useSearchParams();
	const tab: InboxTab = params.get("tab") === "sent" ? "sent" : "conversations";
	const selectedId = Number(params.get("c")) || null;
	const [refreshKey, setRefreshKey] = React.useState(0);
	const [newOpen, setNewOpen] = React.useState(false);
	const [muted, setMuted] = React.useState(isNotificationSoundMuted);

	const update = React.useCallback(
		(changes: Record<string, string | null>) => {
			setParams((prev) => {
				const next = new URLSearchParams(prev);
				for (const [k, v] of Object.entries(changes)) {
					if (v === null) next.delete(k);
					else next.set(k, v);
				}
				return next;
			});
		},
		[setParams],
	);
	const refreshList = React.useCallback(() => setRefreshKey((k) => k + 1), []);

	function toggleMuted() {
		setNotificationSoundMuted(!muted);
		setMuted(!muted);
	}
	const soundLabel = textGet(
		muted ? "whatsapp.inbox.sound.unmute" : "whatsapp.inbox.sound.mute",
	);

	return (
		<div className="grid gap-4">
			<PageHeader
				title={<Text uuid="whatsapp.inbox.title" />}
				description={<Text uuid="whatsapp.inbox.description" />}
				actions={
					<>
						<Button
							variant="outline"
							size="icon"
							onClick={toggleMuted}
							aria-pressed={!muted}
							aria-label={soundLabel}
							title={soundLabel}
						>
							{muted ? <VolumeX /> : <Volume2 />}
						</Button>
						<Button onClick={() => setNewOpen(true)}>
							<MessageSquarePlus />
							<Text uuid="whatsapp.new_message.open" />
						</Button>
					</>
				}
			/>
			<TemplateMessageDialog
				open={newOpen}
				onOpenChange={setNewOpen}
				onSent={({ conversation }) => {
					update({ tab: null, c: String(conversation.id) });
					refreshList();
				}}
			/>
			<Tabs
				value={tab}
				onValueChange={(v) => update({ tab: v === "sent" ? "sent" : null })}
			>
				<TabsList>
					<TabsTrigger value="conversations">
						<Text uuid="whatsapp.inbox.tab.conversations" />
					</TabsTrigger>
					<TabsTrigger value="sent">
						<Text uuid="whatsapp.inbox.tab.sent" />
					</TabsTrigger>
				</TabsList>
				<TabsContent value="conversations">
					<div className="grid h-[calc(100dvh-15rem)] min-h-[28rem] overflow-hidden rounded-lg border bg-card md:grid-cols-[20rem_1fr]">
						<ConversationList
							selectedId={selectedId}
							onSelect={(id) => update({ c: String(id) })}
							refreshKey={refreshKey}
							className={cn("md:border-r", selectedId && "hidden md:flex")}
						/>
						{selectedId ? (
							<ConversationThread
								key={selectedId}
								conversationId={selectedId}
								onBack={() => update({ c: null })}
								onChanged={refreshList}
							/>
						) : (
							<div className="hidden flex-col items-center justify-center gap-2 p-8 text-center text-sm text-muted-foreground md:flex">
								<MessagesSquare className="h-10 w-10" />
								<Text uuid="whatsapp.inbox.thread.pick" />
							</div>
						)}
					</div>
				</TabsContent>
				<TabsContent value="sent">
					<SentMessages />
				</TabsContent>
			</Tabs>
		</div>
	);
}
