import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { getMessages, type MessageMap } from "@/api/i18n-service";
import type { SupportedLocale } from "@/config/zod-i18n";

type UIMessageState = {
	messages: MessageMap;
	lang: SupportedLocale | undefined;
	version: string;
	clean: () => void;
	setMessages: (messages: MessageMap) => void;
	setLang: (lang: SupportedLocale) => void;
	setMessagesVersion: (version: string) => void;
	fetchMessages: (lang: SupportedLocale) => Promise<void>;
};

const persistMessage = persist<UIMessageState>(
	(set) => ({
		messages: {},
		lang: "es",
		version: "",
		clean() {
			set({ messages: {} });
		},
		setMessages(messages) {
			set({ messages });
		},
		setLang(lang) {
			set({ lang });
		},
		setMessagesVersion(version) {
			set({ version });
		},
		fetchMessages: async (lang) => {
			const result = await getMessages(lang);
			if (result.success) {
				const messagesRecord = result.data.reduce<MessageMap>(
					(acc, current) => {
						acc[current.key] = current.value;
						return acc;
					},
					{},
				);
				set({ messages: messagesRecord });
			} else {
				console.error("Failed to fetch messages", result.data);
			}
		},
	}),
	{
		name: "messages",
		storage: createJSONStorage(() => localStorage),
	},
);

export const useMessageStore = create(persistMessage);
