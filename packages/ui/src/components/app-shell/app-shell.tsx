import { Menu, PanelLeft, PanelLeftClose, X } from "lucide-react";
import type React from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation } from "react-router";
import { useUiText } from "../../context/text-context";
import { useViewport } from "../../hooks/use-viewport";
import { cn } from "../../lib/utils";
import { useSidebarStore } from "../../stores/sidebar-store";
import { Button } from "../button";
import NavAccordion from "../nav/nav-accordion";
import NavItem from "../nav/nav-item";
import { AppShellContext, type AppShellState } from "./app-shell-context";

type Icon = React.ComponentType<{ className?: string }>;

export type NavLinkEntry = {
	label: string;
	href: string;
	icon: Icon;
	/** Highlight while its href is the current page (default true). */
	matchActive?: boolean;
};
export type NavGroupEntry = {
	label: string;
	icon: Icon;
	accordionItems: NavLinkEntry[];
};
export type NavEntry = NavLinkEntry | NavGroupEntry;

interface AppShellProps {
	brand: {
		icon: React.ComponentType<{ className?: string; strokeWidth?: number }>;
		name: React.ReactNode;
	};
	nav: NavEntry[];
	/** Entries pinned to the bottom of the sidebar (settings, help). */
	footerNav?: NavEntry[];
	/** Page title in the top bar. */
	title: React.ReactNode;
	/** Right side of the top bar (language, notifications, user menu). */
	actions?: React.ReactNode;
	/** Full-width strip under the top bar (e.g. a subscription warning). */
	banner?: React.ReactNode;
	children: React.ReactNode;
}

const DRAWER_ID = "app-shell-nav";

const renderEntry = (entry: NavEntry) =>
	"accordionItems" in entry && entry.accordionItems ? (
		<NavAccordion {...entry} key={entry.label} />
	) : (
		<NavItem {...(entry as NavLinkEntry)} key={entry.label} />
	);

/**
 * The frame of every signed-in page: sidebar, top bar and scrolling content.
 * It keeps clear of the iPhone notch and home indicator (`safe-area-inset-*`;
 * the apps set `viewport-fit=cover` and pad `#root` left and right).
 * On desktop the sidebar is a rail the user expands or collapses (remembered);
 * on phones it is a drawer over the content that closes on navigation, on
 * Escape, on the backdrop, or when the screen grows to desktop.
 */
export function AppShell({
	brand,
	nav,
	footerNav = [],
	title,
	actions,
	banner,
	children,
}: AppShellProps) {
	const { textGet } = useUiText();
	const { isPhone } = useViewport();
	const { pathname } = useLocation();
	const railOpen = useSidebarStore((s) => s.isOpen);
	const setRailOpen = useSidebarStore((s) => s.setOpen);

	// The drawer is open "on" a page: navigating away or leaving phone width
	// closes it without an effect to keep in sync.
	const [drawerOpenAt, setDrawerOpenAt] = useState<string | null>(null);
	const drawerOpen = isPhone && drawerOpenAt === pathname;
	const expanded = isPhone || railOpen;

	const drawerRef = useRef<HTMLElement>(null);
	const menuButtonRef = useRef<HTMLButtonElement>(null);

	useEffect(() => {
		if (!drawerOpen) return;
		const menuButton = menuButtonRef.current;
		drawerRef.current?.focus();
		const onKeyDown = (event: KeyboardEvent) => {
			if (event.key === "Escape") setDrawerOpenAt(null);
		};
		document.addEventListener("keydown", onKeyDown);
		return () => {
			document.removeEventListener("keydown", onKeyDown);
			menuButton?.focus();
		};
	}, [drawerOpen]);

	const shell: AppShellState = useMemo(
		() => ({
			expanded,
			expand: () => {
				if (!isPhone) setRailOpen(true);
			},
			closeDrawer: () => setDrawerOpenAt(null),
		}),
		[expanded, isPhone, setRailOpen],
	);

	const BrandIcon = brand.icon;

	return (
		<AppShellContext.Provider value={shell}>
			<div className="flex h-dvh overflow-hidden bg-background">
				<aside
					id={DRAWER_ID}
					ref={drawerRef}
					tabIndex={-1}
					aria-label={textGet("shell.nav")}
					// On phones the drawer is a modal dialog over the page.
					{...(isPhone && { role: "dialog", "aria-modal": drawerOpen })}
					// A closed drawer is off-screen: keep it out of the tab order too.
					inert={isPhone && !drawerOpen}
					className={cn(
						"flex shrink-0 flex-col border-r border-border bg-sidebar pb-[env(safe-area-inset-bottom)] outline-none transition-[width,translate] duration-300 ease-out motion-reduce:transition-none",
						isPhone
							? cn(
									"fixed inset-y-0 left-0 z-50 w-72 max-w-[85vw] pl-[env(safe-area-inset-left)] shadow-xl",
									!drawerOpen && "-translate-x-full",
								)
							: railOpen
								? "w-64"
								: "w-16",
					)}
				>
					<div className="flex h-[calc(4rem+env(safe-area-inset-top))] shrink-0 items-center justify-center gap-2 overflow-hidden border-b border-sidebar-border px-4 pt-[env(safe-area-inset-top)]">
						{expanded && (
							<div className="flex min-w-0 flex-1 items-center gap-2">
								<div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary">
									<BrandIcon
										className="h-5 w-5 text-primary-foreground"
										strokeWidth={1.75}
									/>
								</div>
								<span className="min-w-0 truncate text-lg font-semibold text-sidebar-foreground">
									{brand.name}
								</span>
							</div>
						)}
						{isPhone ? (
							<Button
								variant="ghost"
								size="icon"
								onClick={() => setDrawerOpenAt(null)}
								aria-label={textGet("shell.menu.close")}
								className="size-10 text-sidebar-foreground hover:bg-sidebar-accent"
							>
								<X className="h-5 w-5" />
							</Button>
						) : (
							<Button
								variant="ghost"
								size="icon"
								onClick={() => setRailOpen(!railOpen)}
								aria-label={textGet(
									railOpen ? "shell.rail.collapse" : "shell.rail.expand",
								)}
								aria-expanded={railOpen}
								className="size-9 text-sidebar-foreground hover:bg-sidebar-accent"
							>
								{railOpen ? (
									<PanelLeftClose className="h-5 w-5" />
								) : (
									<PanelLeft className="h-5 w-5" />
								)}
							</Button>
						)}
					</div>

					<nav className="flex-1 space-y-1 overflow-x-hidden overflow-y-auto overscroll-contain p-2">
						{nav.map(renderEntry)}
					</nav>

					{footerNav.length > 0 && (
						<div className="shrink-0 space-y-1 overflow-hidden border-t border-sidebar-border p-2 py-4">
							{footerNav.map(renderEntry)}
						</div>
					)}
				</aside>

				{drawerOpen && (
					<div
						aria-hidden
						className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm animate-in fade-in-0 duration-200 motion-reduce:animate-none"
						onClick={() => setDrawerOpenAt(null)}
					/>
				)}

				<div className="flex min-w-0 flex-1 flex-col overflow-hidden">
					<header className="flex h-[calc(4rem+env(safe-area-inset-top))] shrink-0 items-center justify-between gap-2 border-b border-border bg-card px-2 pt-[env(safe-area-inset-top)] sm:px-4 md:px-6">
						<div className="flex min-w-0 items-center gap-1">
							{isPhone && (
								<Button
									ref={menuButtonRef}
									variant="ghost"
									size="icon"
									className="size-10"
									onClick={() => setDrawerOpenAt(pathname)}
									aria-label={textGet("shell.menu.open")}
									aria-expanded={drawerOpen}
									aria-controls={DRAWER_ID}
								>
									<Menu className="h-5 w-5" />
								</Button>
							)}
							<span className="truncate text-lg font-semibold">{title}</span>
						</div>
						{actions && (
							<div className="flex shrink-0 items-center gap-1 sm:gap-2">
								{actions}
							</div>
						)}
					</header>

					{banner}

					<main className="relative flex-1 overflow-auto p-4 pb-[max(1rem,env(safe-area-inset-bottom))] md:p-6 md:pb-[max(1.5rem,env(safe-area-inset-bottom))]">
						{children}
					</main>
				</div>
			</div>
		</AppShellContext.Provider>
	);
}
