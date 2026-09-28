import React from "react";
import { toast } from "sonner";
import { useLanguage } from "./language-context";
import { useMessageStore } from "./message-store";

/**
 * Keeps the UI messages loaded and current. The cached messages (localStorage)
 * render right away; on load and on every language change they are
 * revalidated in the background by content hash (ETag / If-None-Match), so
 * new keys show up on the next load without refetching unchanged messages.
 */
export function useMessages() {
	const fetchMessages = useMessageStore((s) => s.fetchMessages);
	const { currentLanguage } = useLanguage();

	React.useEffect(() => {
		fetchMessages(currentLanguage).catch(() => {
			const { lang, messages } = useMessageStore.getState();
			// A failed background revalidation leaves usable messages: stay quiet.
			if (lang === currentLanguage && Object.keys(messages).length > 0) return;
			// Can't be an i18n key: the messages are what failed to load.
			toast.error("Error al cargar los mensajes del servicio.");
		});
	}, [currentLanguage, fetchMessages]);
}
