import { UiTextProvider } from "@pengi/ui";
import type { ReactNode } from "react";
import { useText } from "@/hooks/use-text";

export function AppTextBridge({ children }: { children: ReactNode }) {
	const { textGet } = useText();
	return <UiTextProvider value={{ textGet }}>{children}</UiTextProvider>;
}
