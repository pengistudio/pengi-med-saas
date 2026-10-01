import { createContext, type ReactNode, useContext } from "react";

export interface TextApi {
	/** The message for `key`, with `{name}` placeholders filled from `values`. */
	textGet: (key: string, values?: Record<string, string | number>) => string;
	/** Current UI language; forms revalidate when it changes to refresh their messages. */
	language?: string;
}

const TextContext = createContext<TextApi | null>(null);

export function UiTextProvider({
	value,
	children,
}: {
	value: TextApi;
	children: ReactNode;
}) {
	return <TextContext.Provider value={value}>{children}</TextContext.Provider>;
}

export function useUiText() {
	const context = useContext(TextContext);
	if (!context) {
		throw new Error("useUiText must be used within a UiTextProvider");
	}
	return context;
}
