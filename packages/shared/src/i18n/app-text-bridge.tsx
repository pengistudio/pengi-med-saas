import { UiTextProvider } from "@pengi/ui";
import type { ReactNode } from "react";
import { useLanguage } from "./language-context";
import { useText } from "./use-text";

export function AppTextBridge({ children }: { children: ReactNode }) {
	const { textGet } = useText();
	const { currentLanguage } = useLanguage();
	return (
		<UiTextProvider value={{ textGet, language: currentLanguage }}>
			{children}
		</UiTextProvider>
	);
}
