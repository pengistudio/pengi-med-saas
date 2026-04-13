import React, { Suspense } from "react";
import { RouterProvider } from "react-router";
import { toast } from "sonner";
import { useLanguage } from "./contexts/language-context";
import { router } from "./routes/routes";
import { useMessageStore } from "./store/message-store";

export function App() {
	const { fetchMessages, messages, setLang } = useMessageStore();
	const { currentLanguage } = useLanguage();
	const prevLanguage = React.useRef(currentLanguage);

	const messageLength = React.useMemo(
		() => Object.keys(messages).length,
		[messages],
	);

	React.useEffect(() => {
		if (messageLength === 0 || prevLanguage.current !== currentLanguage) {
			prevLanguage.current = currentLanguage;
			fetchMessages(currentLanguage)
				.then(() => {
					setLang(currentLanguage);
				})
				.catch(() => {
					toast.error("Error al cargar los mensajes del servicio.");
				});
		}
	}, [messageLength, fetchMessages, currentLanguage, setLang]);

	return (
		<Suspense fallback={null}>
			<RouterProvider router={router} />
		</Suspense>
	);
}

export default App;
