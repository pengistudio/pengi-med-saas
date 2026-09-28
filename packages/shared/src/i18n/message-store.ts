import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { getMessages, type MessageMap } from "./messages-service";
import type { SupportedLocale } from "./zod-i18n";

type UIMessageState = {
	messages: MessageMap;
	/** Language of `messages`. */
	lang: SupportedLocale | undefined;
	/** ETag (hash of the content) of `messages`, sent back as If-None-Match. */
	etag: string | undefined;
	clean: () => void;
	setMessages: (messages: MessageMap) => void;
	setLang: (lang: SupportedLocale) => void;
	/**
	 * Loads the messages of `lang`. When the cache holds that language it is
	 * revalidated with its ETag: a 304 keeps it, a 200 replaces it. Rejects when
	 * the request fails.
	 */
	fetchMessages: (lang: SupportedLocale) => Promise<void>;
};

// The request in flight: a second call for the same language reuses it (e.g.
// StrictMode's double effect), and a call for another language supersedes it
// so a slow, older response can't overwrite the newer language.
let inFlight: { lang: SupportedLocale; promise: Promise<void> } | undefined;

const persistMessage = persist<UIMessageState>(
	(set, get) => ({
		messages: {},
		lang: "es",
		etag: undefined,
		clean() {
			set({ messages: {}, etag: undefined });
		},
		setMessages(messages) {
			set({ messages });
		},
		setLang(lang) {
			set({ lang });
		},
		fetchMessages(lang) {
			if (inFlight?.lang === lang) return inFlight.promise;

			const state = get();
			const cached =
				state.lang === lang && Object.keys(state.messages).length > 0;
			const promise: Promise<void> = getMessages(
				lang,
				cached ? state.etag : undefined,
			)
				.then((result) => {
					if (inFlight?.promise !== promise) return;
					if (result.status === "fresh") {
						set({ messages: result.messages, etag: result.etag, lang });
					} else if (!cached) {
						throw new Error("@pengi/shared: 304 without cached messages");
					}
				})
				.finally(() => {
					if (inFlight?.promise === promise) inFlight = undefined;
				});
			inFlight = { lang, promise };
			return promise;
		},
	}),
	{
		name: "messages",
		storage: createJSONStorage(() => localStorage),
		// v0 kept a build id (`version`) instead of the ETag.
		version: 1,
		migrate: (persisted) => {
			const { version: _build, ...rest } = persisted as UIMessageState & {
				version?: string;
			};
			return { ...rest, etag: undefined } as UIMessageState;
		},
	},
);

export const useMessageStore = create(persistMessage);
