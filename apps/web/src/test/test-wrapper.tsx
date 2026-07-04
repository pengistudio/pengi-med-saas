import { render } from "@testing-library/react";
import type React from "react";
import type { ReactNode } from "react";
import { AppTextBridge } from "@/components/app-text-bridge";
import { LanguageProvider } from "@/contexts/language-context";

/**
 * Wrapper para tests que proporcionan contextos y providers necesarios
 */
export const TestWrapper = ({ children }: { children: ReactNode }) => {
	return (
		<LanguageProvider>
			<AppTextBridge>{children}</AppTextBridge>
		</LanguageProvider>
	);
};

/**
 * Custom render que incluye todos los providers
 */
export function renderWithProviders(
	ui: React.ReactElement,
	options?: Record<string, unknown>,
) {
	const Wrapper = ({ children }: { children: ReactNode }) => (
		<TestWrapper>{children}</TestWrapper>
	);

	return render(ui, { wrapper: Wrapper, ...options });
}
