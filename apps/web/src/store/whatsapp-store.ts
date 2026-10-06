import { create } from "zustand";
import { getWhatsAppUnreadCount } from "@/api/whatsapp-service";

/** The inbox unread counter, shared by the nav badge and the inbox page. */
interface WhatsAppStore {
	/** Conversations with unread messages (the nav badge). */
	unreadConversations: number;
	/** Unread messages across conversations. */
	unreadMessages: number;
	setUnread: (conversations: number, messages: number) => void;
}

export const useWhatsAppStore = create<WhatsAppStore>((set) => ({
	unreadConversations: 0,
	unreadMessages: 0,
	setUnread: (unreadConversations, unreadMessages) =>
		set({ unreadConversations, unreadMessages }),
}));

export const selectUnreadConversations = (s: WhatsAppStore) =>
	s.unreadConversations;

/** Re-reads the counter (polled by the layout, called after marking read). */
export async function refreshWhatsAppUnread(): Promise<void> {
	const res = await getWhatsAppUnreadCount();
	if (res.success) {
		useWhatsAppStore
			.getState()
			.setUnread(res.data.conversations, res.data.unread_count);
	}
}
