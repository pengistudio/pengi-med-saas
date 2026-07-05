import { createContext, type ReactNode, useContext } from "react";

export interface TextApi {
	textGet: (key: string) => string;
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
