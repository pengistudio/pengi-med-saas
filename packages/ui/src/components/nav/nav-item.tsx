import type React from "react";
import { Link, useLocation } from "react-router";
import { cn } from "../../lib/utils";
import { useAppShell } from "../app-shell/app-shell-context";

type Props = {
	label: string;
	href: string;
	icon: React.ComponentType<{ className?: string }>;
	/** Highlight the item while its href is the current page (default true). */
	matchActive?: boolean;
};

const NavItem = ({ label, href, icon, matchActive = true }: Props) => {
	const { expanded, closeDrawer } = useAppShell();
	const location = useLocation();

	const isActive = matchActive && location.pathname === href;

	const Icon = icon;
	return (
		<Link
			to={href}
			title={!expanded ? label : undefined}
			aria-current={isActive ? "page" : undefined}
			className={cn(
				"flex min-h-10 items-center rounded-lg px-3 py-2 transition-all duration-300 overflow-hidden outline-none focus-visible:ring-3 focus-visible:ring-ring/50 motion-reduce:transition-none",
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
		</Link>
	);
};

export default NavItem;
