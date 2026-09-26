import { createContext, type ReactNode, useContext, useState } from "react";
import { useMessageStore } from "./message-store";
import { type SupportedLocale, updateZodLocale } from "./zod-i18n";

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

	// zod's locale is switched before the state changes, not in an effect:
	// children's effects run before their parent's, so a form re-validating on
	// the language change would otherwise still get the old language's messages.
	const [currentLanguage, setCurrentLanguage] = useState<SupportedLocale>(
		() => {
			const initial = lang ?? "es";
			updateZodLocale(initial);
			return initial;
		},
	);

	const changeLanguage = (lang: SupportedLocale) => {
		updateZodLocale(lang);
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
