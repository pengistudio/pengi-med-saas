import { AppTextBridge, LanguageProvider } from "@pengi/shared";
import { render } from "@testing-library/react";
import type React from "react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";

/** Providers every backoffice screen needs: i18n context and a router. */
export const TestWrapper = ({
	children,
	initialEntries = ["/"],
}: {
	children: ReactNode;
	initialEntries?: string[];
}) => (
	<LanguageProvider>
		<AppTextBridge>
			<MemoryRouter initialEntries={initialEntries}>{children}</MemoryRouter>
		</AppTextBridge>
	</LanguageProvider>
);

export function renderWithProviders(
	ui: React.ReactElement,
	{ initialEntries }: { initialEntries?: string[] } = {},
) {
	const Wrapper = ({ children }: { children: ReactNode }) => (
		<TestWrapper initialEntries={initialEntries}>{children}</TestWrapper>
	);
	return render(ui, { wrapper: Wrapper });
}
