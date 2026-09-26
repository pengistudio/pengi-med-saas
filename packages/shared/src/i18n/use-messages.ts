import React from "react";
import { toast } from "sonner";
import { useLanguage } from "./language-context";
import { useMessageStore } from "./message-store";

/**
 * Keeps the UI messages loaded: fetches them when there are none, when the
 * language changes, and when the app was rebuilt (a new build may add keys the
 * cached messages in localStorage don't have).
 *
 * @param appVersion changes on every build (e.g. Vite's __APP_VERSION__).
 */
export function useMessages(appVersion: string) {
	const {
		fetchMessages,
		messages,
		lang,
		setLang,
		version,
		setMessagesVersion,
	} = useMessageStore();
	const { currentLanguage } = useLanguage();
	// The language+build already requested: storing the messages re-renders
	// before the version is saved, which must not trigger a second fetch.
	const requested = React.useRef<string | null>(null);
	const hasMessages = Object.keys(messages).length > 0;

	React.useEffect(() => {
		const wanted = `${currentLanguage}@${appVersion}`;
		const upToDate =
			hasMessages && lang === currentLanguage && version === appVersion;
		if (upToDate || requested.current === wanted) return;

		requested.current = wanted;
		fetchMessages(currentLanguage)
			.then(() => {
				setLang(currentLanguage);
				setMessagesVersion(appVersion);
			})
			.catch(() => {
				requested.current = null;
				// Can't be an i18n key: the messages are what failed to load.
				toast.error("Error al cargar los mensajes del servicio.");
			});
	}, [
		hasMessages,
		lang,
		version,
		currentLanguage,
		appVersion,
		fetchMessages,
		setLang,
		setMessagesVersion,
	]);
}
