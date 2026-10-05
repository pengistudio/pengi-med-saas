import type React from "react";
import { Link, useLocation } from "react-router";
import { cn } from "../../lib/utils";
import { useAppShell } from "../app-shell/app-shell-context";
import { NavBadge } from "./nav-badge";

type Props = {
	label: string;
	href: string;
	icon: React.ComponentType<{ className?: string }>;
	/** Highlight the item while its href is the current page (default true). */
	matchActive?: boolean;
	/** Overrides the exact-path check when the parent computes the active link. */
	active?: boolean;
	/** Pending count shown next to the label; hidden at 0, capped at 99+. */
	badge?: number;
};

const NavItem = ({
	label,
	href,
	icon,
	matchActive = true,
	active,
	badge,
}: Props) => {
	const { expanded, closeDrawer } = useAppShell();
	const location = useLocation();

	const isActive = active ?? (matchActive && location.pathname === href);

	const Icon = icon;
	return (
		<Link
			to={href}
			viewTransition
			title={!expanded ? label : undefined}
			aria-current={isActive ? "page" : undefined}
			className={cn(
				"relative flex min-h-10 items-center rounded-lg px-3 py-2 transition-all duration-300 overflow-hidden outline-none focus-visible:ring-3 focus-visible:ring-ring/50 motion-reduce:transition-none",
				expanded ? "gap-3" : "gap-0",
				isActive
					? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
					: "text-sidebar-foreground hover:bg-sidebar-accent",
			)}
			// Tapping the page you are on doesn't change the route, so close here too.
			onClick={closeDrawer}
		>
			<Icon className={cn("h-5 w-5 shrink-0", isActive && "text-primary")} />
			<span
				className={cn(
					"transition-all duration-300 truncate motion-reduce:transition-none",
					expanded
						? "opacity-100 w-auto"
						: "opacity-0 w-0 overflow-hidden ml-0",
				)}
			>
				{label}
			</span>
			<NavBadge
				count={badge}
				className={
					expanded ? "ml-auto" : "absolute top-0.5 right-1 h-4 min-w-4"
				}
			/>
		</Link>
	);
};

export default NavItem;
