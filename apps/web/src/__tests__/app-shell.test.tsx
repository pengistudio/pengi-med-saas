import { AppShell, UiTextProvider } from "@pengi/ui";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FileText, Home } from "lucide-react";
import { MemoryRouter, Outlet, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
	createNavItems,
	filterNavItems,
	pickBottomNav,
} from "@/config/nav-config";
import { PERMISSIONS } from "@/lib/constants";

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

describe("AppShell sections", () => {
	afterEach(() => vi.unstubAllGlobals());

	const renderNav = (
		nav: { label: string; href: string; section?: string }[],
		path = "/",
	) => {
		stubViewport({ phone: false });
		return render(
			<UiTextProvider value={{ textGet: (key) => key }}>
				<MemoryRouter initialEntries={[path]}>
					<AppShell
						brand={{ icon: Home, name: "Clínica" }}
						nav={nav.map((n) => ({ ...n, icon: Home }))}
						matchNested
						title="T"
					>
						<p>page</p>
					</AppShell>
				</MemoryRouter>
			</UiTextProvider>,
		);
	};

	it("shows each section title once, above its items", () => {
		renderNav([
			{ label: "Inicio", href: "/" },
			{ label: "Agenda", href: "/a", section: "Atención" },
			{ label: "Pacientes", href: "/p", section: "Atención" },
			{ label: "Equipo", href: "/t", section: "Administración" },
		]);
		expect(screen.getAllByText("Atención")).toHaveLength(1);
		expect(screen.getByText("Administración")).toBeInTheDocument();
	});

	it("hides a section title when all its items are filtered out", () => {
		const granted = [PERMISSIONS.BILLING.PERMISSION_READ_BILLING];
		const nav = filterNavItems(
			createNavItems((key) => key),
			(needed) => needed.every((p) => granted.includes(p)),
			{},
		).filter((i) => !i.isBottom);
		renderNav(
			nav.map((i) => ({
				label: i.label,
				href: i.href ?? "/",
				section: i.section,
			})),
		);
		expect(screen.queryByText("nav.section.care")).not.toBeInTheDocument();
		expect(screen.queryByText("nav.section.admin")).not.toBeInTheDocument();
		expect(screen.getByText("nav.section.finance")).toBeInTheDocument();
	});

	it("highlights the longest matching link on nested pages", () => {
		renderNav(
			[
				{ label: "Pacientes", href: "/clinical" },
				{ label: "Exámenes", href: "/clinical/exam-orders" },
			],
			"/clinical/exam-orders/1",
		);
		expect(screen.getByRole("link", { name: "Exámenes" })).toHaveAttribute(
			"aria-current",
			"page",
		);
		expect(screen.getByRole("link", { name: "Pacientes" })).not.toHaveAttribute(
			"aria-current",
		);
	});
});

describe("AppShell bottom bar", () => {
	afterEach(() => vi.unstubAllGlobals());

	const links = (...hrefs: string[]) =>
		hrefs.map((href) => ({ label: `L${href}`, href, icon: Home }));

	const renderBar = ({
		phone = true,
		path = "/",
		bottomNav = links("/", "/clinical"),
		badge,
	}: {
		phone?: boolean;
		path?: string;
		bottomNav?: ReturnType<typeof links>;
		badge?: number;
	} = {}) => {
		stubViewport({ phone });
		return render(
			<UiTextProvider value={{ textGet: (key) => key }}>
				<MemoryRouter initialEntries={[path]}>
					<AppShell
						brand={{ icon: Home, name: "C" }}
						nav={links("/", "/clinical", "/other")}
						matchNested
						bottomNav={bottomNav.map((l) =>
							l.href === "/clinical" ? { ...l, badge } : l,
						)}
						title="T"
					>
						<p>page</p>
					</AppShell>
				</MemoryRouter>
			</UiTextProvider>,
		);
	};

	const bar = () =>
		screen.queryByRole("navigation", { name: "shell.nav.primary" });

	it("renders only on phone width", () => {
		renderBar({ phone: false });
		expect(bar()).not.toBeInTheDocument();
	});

	it("marks the current tab and not Más", () => {
		renderBar({ path: "/clinical/patients/3" });
		expect(
			within(bar() as HTMLElement).getByRole("link", { name: "L/clinical" }),
		).toHaveAttribute("aria-current", "page");
		expect(screen.getByRole("button", { name: "nav.more" })).toHaveClass(
			"text-muted-foreground",
		);
	});

	it("highlights Más when the route is not one of the tabs, and opens the drawer", async () => {
		const user = userEvent.setup();
		renderBar({ path: "/other" });
		const more = screen.getByRole("button", { name: "nav.more" });
		expect(more).toHaveClass("text-primary");
		await user.click(more);
		expect(more).toHaveAttribute("aria-expanded", "true");
	});

	it("hides the badge at 0 and caps it at 99+", () => {
		const { unmount } = renderBar({ badge: 0 });
		expect(
			within(bar() as HTMLElement).queryByText("0"),
		).not.toBeInTheDocument();
		unmount();
		renderBar({ badge: 250 });
		expect(within(bar() as HTMLElement).getByText("99+")).toBeInTheDocument();
	});
});

// Permissions of the canonical roles (role_data.RolePermissionMatrix in the API).
const ROLE_PERMISSIONS = {
	doctor: [
		"READ_PATIENT",
		"CREATE_PATIENT",
		"UPDATE_PATIENT",
		"READ_MEDICAL_RECORD",
		"CREATE_MEDICAL_RECORD",
		"UPDATE_MEDICAL_RECORD",
		"READ_EXAM_ORDER",
		"CREATE_EXAM_ORDER",
		"READ_APPOINTMENT",
		"MANAGE_APPOINTMENT",
		"RECORD_VITAL_SIGNS",
		"READ_KANBAN",
	],
	recepcionista: [
		"READ_PATIENT",
		"CREATE_PATIENT",
		"UPDATE_PATIENT",
		"UPLOAD_PATIENT_ATTACHMENT",
		"READ_EXAM_ORDER",
		"UPLOAD_EXAM_RESULTS",
		"READ_APPOINTMENT",
		"MANAGE_APPOINTMENT",
		"RECORD_VITAL_SIGNS",
		"READ_BILLING",
		"CREATE_BILLING",
		"UPDATE_BILLING",
		"READ_KANBAN",
	],
	contador: [
		"READ_BILLING",
		"CREATE_BILLING",
		"UPDATE_BILLING",
		"DELETE_BILLING",
		"MANAGE_SRI_SETTINGS",
		"READ_PATIENT",
	],
};

const visibleNav = (granted: string[]) =>
	filterNavItems(
		createNavItems((key) => key),
		(needed) => needed.every((p) => granted.includes(p)),
		{},
	);

describe("nav per role", () => {
	const hrefs = (granted: string[]) =>
		visibleNav(granted)
			.filter((i) => !i.isBottom)
			.map((i) => i.href);

	it("gives the front desk the agenda, waiting room, patients and exams", () => {
		expect(hrefs(ROLE_PERMISSIONS.recepcionista)).toEqual(
			expect.arrayContaining([
				"/clinical/appointments",
				"/clinical/waiting-room",
				"/clinical",
				"/clinical/exam-orders",
				"/billing",
			]),
		);
	});

	it("keeps the accountant out of care: READ_PATIENT alone is for invoicing", () => {
		const nav = hrefs(ROLE_PERMISSIONS.contador);
		expect(nav).not.toContain("/clinical/appointments");
		expect(nav).not.toContain("/clinical/waiting-room");
		expect(nav).not.toContain("/clinical");
		expect(nav).toContain("/billing");
	});

	it("requires every permission an item lists", () => {
		const items = filterNavItems(
			[
				{ label: "a", href: "/a", icon: Home, permission: ["X", "Y"] },
				{ label: "b", href: "/b", icon: Home, permission: "X" },
			],
			(needed) => needed.every((p) => ["X"].includes(p)),
			{},
		);
		expect(items.map((i) => i.href)).toEqual(["/b"]);
	});
});

describe("pickBottomNav", () => {
	const pick = (granted: string[]) =>
		pickBottomNav(visibleNav(granted), (key) => key).map((i) => i.href);

	it("gives a doctor the care tabs", () => {
		expect(pick(ROLE_PERMISSIONS.doctor)).toEqual([
			"/",
			"/clinical/appointments",
			"/clinical",
			"/clinical/exam-orders",
		]);
	});

	it("gives the front desk the care tabs", () => {
		expect(pick(ROLE_PERMISSIONS.recepcionista)).toEqual([
			"/",
			"/clinical/appointments",
			"/clinical",
			"/clinical/exam-orders",
		]);
	});

	it("gives an accountant the billing tabs", () => {
		expect(pick(ROLE_PERMISSIONS.contador)).toEqual([
			"/",
			"/billing",
			"/billing/credit-notes",
			"/billing/catalog-items",
		]);
	});
});
