import {
	createContext,
	type ReactNode,
	useContext,
	useEffect,
	useState,
} from "react";
import { useMessageStore } from "@/store/message-store";
import { type SupportedLocale, updateZodLocale } from "../config/zod-i18n";

type LanguageContextType = {
	currentLanguage: SupportedLocale;
	changeLanguage: (lang: SupportedLocale) => void;
};

const LanguageContext = createContext<LanguageContextType | undefined>(
	undefined,
);

export function LanguageProvider({
	children,
}: {
	children: ReactNode;
	initialLang?: SupportedLocale;
}) {
	const { lang } = useMessageStore();

	const [currentLanguage, setCurrentLanguage] = useState<SupportedLocale>(
		lang ?? "es",
	);

	useEffect(() => {
		updateZodLocale(currentLanguage);
	}, [currentLanguage]);

	const changeLanguage = (lang: SupportedLocale) => {
		setCurrentLanguage(lang);
	};

	return (
		<LanguageContext.Provider value={{ currentLanguage, changeLanguage }}>
			{children}
		</LanguageContext.Provider>
	);
}

export function useLanguage() {
	const context = useContext(LanguageContext);
	if (context === undefined) {
		throw new Error("useLanguage must be used within a LanguageProvider");
	}
	return context;
}
