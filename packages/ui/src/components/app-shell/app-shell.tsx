import { Ellipsis, Menu, PanelLeft, PanelLeftClose, X } from "lucide-react";
import type React from "react";
import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import { Link, useLocation } from "react-router";
import { useUiText } from "../../context/text-context";
import { useViewport } from "../../hooks/use-viewport";
import { cn } from "../../lib/utils";
import { useSidebarStore } from "../../stores/sidebar-store";
import { Button } from "../button";
import NavAccordion from "../nav/nav-accordion";
import { NavBadge } from "../nav/nav-badge";
import NavItem from "../nav/nav-item";
import { AppShellContext, type AppShellState } from "./app-shell-context";

type Icon = React.ComponentType<{ className?: string }>;

export type NavLinkEntry = {
	label: string;
	href: string;
	icon: Icon;
	/** Highlight while its href is the current page (default true). */
	matchActive?: boolean;
	/** Title of the section this entry opens; entries following it share it. */
	section?: string;
	/** Pending count next to the label; hidden at 0, shown as "99+" above 99. */
	badge?: number;
};
export type NavGroupEntry = {
	label: string;
	icon: Icon;
	section?: string;
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
	/**
	 * Highlight the link whose href is the longest prefix of the current page
	 * (so `/clinical/exam-orders/1` lights up its list). Off: exact match only.
	 */
	matchNested?: boolean;
	/**
	 * Phone-only tab bar fixed to the bottom: these links plus a "Más" button
	 * that opens the drawer (the full menu). Omit it for no bar.
	 */
	bottomNav?: NavLinkEntry[];
	/** Page title in the top bar. */
	title: React.ReactNode;
	/** Right side of the top bar (language, notifications, user menu). */
	actions?: React.ReactNode;
	/** Full-width strip under the top bar (e.g. a subscription warning). */
	banner?: React.ReactNode;
	children: React.ReactNode;
}

const DRAWER_ID = "app-shell-nav";

const linksOf = (entry: NavEntry): NavLinkEntry[] =>
	"accordionItems" in entry && entry.accordionItems
		? entry.accordionItems
		: [entry as NavLinkEntry];

/** Longest-prefix match of `pathname` among the links, as in `navTitle`. */
function activeHrefFor(entries: NavEntry[], pathname: string) {
	let best: string | undefined;
	for (const link of entries.flatMap(linksOf)) {
		if (link.matchActive === false) continue;
		const hit =
			link.href === "/"
				? pathname === "/"
				: pathname === link.href || pathname.startsWith(`${link.href}/`);
		if (hit && (!best || link.href.length > best.length)) best = link.href;
	}
	return best;
}

const renderEntries = (
	entries: NavEntry[],
	expanded: boolean,
	activeHref: string | undefined,
	matchNested: boolean,
) => {
	let currentSection: string | undefined;
	return entries.map((entry, index) => {
		const title =
			entry.section && entry.section !== currentSection
				? entry.section
				: undefined;
		currentSection = entry.section;
		return (
			<Fragment key={entry.label}>
				{title && !expanded && index > 0 && (
					<hr className="mx-3 my-2 border-0 border-t border-sidebar-border" />
				)}
				{title && (
					<div
						role="presentation"
						className={cn(
							"mt-3 px-3 pt-2 pb-1 text-xs font-medium tracking-wide text-sidebar-foreground/60 uppercase",
							!expanded && "hidden",
						)}
					>
						{title}
					</div>
				)}
				{"accordionItems" in entry && entry.accordionItems ? (
					<NavAccordion {...entry} />
				) : (
					<NavItem
						{...(entry as NavLinkEntry)}
						active={
							matchNested
								? activeHref === (entry as NavLinkEntry).href &&
									(entry as NavLinkEntry).matchActive !== false
								: undefined
						}
					/>
				)}
			</Fragment>
		);
	});
};

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
	matchNested = false,
	bottomNav,
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
	const moreButtonRef = useRef<HTMLButtonElement>(null);
	const openerRef = useRef<HTMLButtonElement | null>(null);
	const showBottomBar = isPhone && !!bottomNav?.length;

	useEffect(() => {
		if (!drawerOpen) return;
		const menuButton = openerRef.current ?? menuButtonRef.current;
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
	const activeHref = useMemo(
		() =>
			matchNested || bottomNav
				? activeHrefFor([...nav, ...footerNav, ...(bottomNav ?? [])], pathname)
				: undefined,
		[matchNested, nav, footerNav, bottomNav, pathname],
	);

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
						{renderEntries(nav, expanded, activeHref, matchNested)}
					</nav>

					{footerNav.length > 0 && (
						<div className="shrink-0 space-y-1 overflow-hidden border-t border-sidebar-border p-2 py-4">
							{renderEntries(footerNav, expanded, activeHref, matchNested)}
						</div>
					)}
				</aside>

				{showBottomBar && bottomNav && (
					<nav
						aria-label={textGet("shell.nav.primary")}
						className="fixed inset-x-0 bottom-0 z-30 flex border-t border-border bg-card pr-[env(safe-area-inset-right)] pb-[env(safe-area-inset-bottom)] pl-[env(safe-area-inset-left)]"
					>
						{bottomNav.map((item) => {
							const Icon = item.icon;
							const isActive = activeHref === item.href;
							return (
								<Link
									key={item.href}
									to={item.href}
									viewTransition
									aria-current={isActive ? "page" : undefined}
									className={cn(
										"relative flex min-h-14 min-w-0 flex-1 flex-col items-center justify-center gap-0.5 px-1 text-[0.6875rem] outline-none focus-visible:ring-3 focus-visible:ring-ring/50",
										isActive
											? "font-medium text-primary"
											: "text-muted-foreground",
									)}
								>
									<Icon className="h-5 w-5 shrink-0" />
									<span className="max-w-full truncate">{item.label}</span>
									<NavBadge
										count={item.badge}
										className="absolute top-1 left-1/2 ml-2 h-4 min-w-4"
									/>
								</Link>
							);
						})}
						<button
							ref={moreButtonRef}
							type="button"
							onClick={() => {
								openerRef.current = moreButtonRef.current;
								setDrawerOpenAt(pathname);
							}}
							aria-expanded={drawerOpen}
							aria-controls={DRAWER_ID}
							className={cn(
								"flex min-h-14 min-w-0 flex-1 flex-col items-center justify-center gap-0.5 px-1 text-[0.6875rem] outline-none focus-visible:ring-3 focus-visible:ring-ring/50",
								!bottomNav.some((i) => i.href === activeHref)
									? "font-medium text-primary"
									: "text-muted-foreground",
							)}
						>
							<Ellipsis className="h-5 w-5 shrink-0" />
							<span className="max-w-full truncate">{textGet("nav.more")}</span>
						</button>
					</nav>
				)}

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
									onClick={() => {
										openerRef.current = menuButtonRef.current;
										setDrawerOpenAt(pathname);
									}}
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

					<main
						className={cn(
							"relative flex-1 overflow-auto p-4 pb-[max(1rem,env(safe-area-inset-bottom))] md:p-6 md:pb-[max(1.5rem,env(safe-area-inset-bottom))]",
							showBottomBar && "pb-[calc(5rem+env(safe-area-inset-bottom))]",
						)}
					>
						{children}
					</main>
				</div>
			</div>
		</AppShellContext.Provider>
	);
}
