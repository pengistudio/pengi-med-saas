import { AppShell, UiTextProvider } from "@pengi/ui";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FileText, Home } from "lucide-react";
import { MemoryRouter, Outlet, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";

function stubViewport({ phone }: { phone: boolean }) {
	vi.stubGlobal("matchMedia", (media: string) => ({
		matches: phone,
		media,
		addEventListener: () => {},
		removeEventListener: () => {},
	}));
}

function renderShell() {
	const Layout = () => (
		<AppShell
			brand={{ icon: Home, name: "Clínica Norte" }}
			nav={[
				{ label: "Inicio", href: "/", icon: Home },
				{ label: "Facturas", href: "/billing", icon: FileText },
			]}
			title="Título"
		>
			<Outlet />
		</AppShell>
	);
	return render(
		<UiTextProvider value={{ textGet: (key) => key }}>
			<MemoryRouter initialEntries={["/"]}>
				<Routes>
					<Route element={<Layout />}>
						<Route path="/" element={<p>home page</p>} />
						<Route path="/billing" element={<p>billing page</p>} />
					</Route>
				</Routes>
			</MemoryRouter>
		</UiTextProvider>,
	);
}

const menuButton = () =>
	screen.getByRole("button", { name: "shell.menu.open" });

describe("AppShell on a phone", () => {
	afterEach(() => vi.unstubAllGlobals());

	it("opens the drawer from the menu button and closes it on Escape", async () => {
		stubViewport({ phone: true });
		const user = userEvent.setup();
		renderShell();

		expect(menuButton()).toHaveAttribute("aria-expanded", "false");
		await user.click(menuButton());
		expect(menuButton()).toHaveAttribute("aria-expanded", "true");
		expect(screen.getByRole("dialog", { name: "shell.nav" })).toHaveFocus();

		await user.keyboard("{Escape}");
		expect(menuButton()).toHaveAttribute("aria-expanded", "false");
		expect(menuButton()).toHaveFocus();
	});

	it("navigates without a reload and closes the drawer", async () => {
		stubViewport({ phone: true });
		const user = userEvent.setup();
		renderShell();

		await user.click(menuButton());
		await user.click(screen.getByRole("link", { name: "Facturas" }));

		expect(screen.getByText("billing page")).toBeInTheDocument();
		expect(menuButton()).toHaveAttribute("aria-expanded", "false");
	});
});

describe("AppShell on a desktop", () => {
	afterEach(() => vi.unstubAllGlobals());

	it("shows a rail the user collapses instead of a menu button", async () => {
		stubViewport({ phone: false });
		const user = userEvent.setup();
		renderShell();

		expect(
			screen.queryByRole("button", { name: "shell.menu.open" }),
		).not.toBeInTheDocument();
		expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

		await user.click(
			screen.getByRole("button", { name: "shell.rail.collapse" }),
		);
		expect(
			screen.getByRole("button", { name: "shell.rail.expand" }),
		).toHaveAttribute("aria-expanded", "false");
	});
});
