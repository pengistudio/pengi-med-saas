import React from "react";
import { RouterProvider } from "react-router";
import { toast } from "sonner";
import { useLanguage } from "./contexts/language-context";
import { router } from "./routes/routes";
import { useMessageStore } from "./store/message-store";

export function App() {
	const { fetchMessages, messages, setLang, version, setMessagesVersion } =
		useMessageStore();
	const { currentLanguage } = useLanguage();
	const prevLanguage = React.useRef(currentLanguage);

	const messageLength = React.useMemo(
		() => Object.keys(messages).length,
		[messages],
	);

	React.useEffect(() => {
		const isNewVersion = version !== __APP_VERSION__;
		if (
			messageLength === 0 ||
			prevLanguage.current !== currentLanguage ||
			isNewVersion
		) {
			prevLanguage.current = currentLanguage;
			fetchMessages(currentLanguage)
				.then(() => {
					setLang(currentLanguage);
					setMessagesVersion(__APP_VERSION__);
				})
				.catch(() => {
					toast.error("Error al cargar los mensajes del servicio.");
				});
		}
	}, [
		messageLength,
		fetchMessages,
		currentLanguage,
		setLang,
		version,
		setMessagesVersion,
	]);

	return <RouterProvider router={router} />;
}

export default App;
