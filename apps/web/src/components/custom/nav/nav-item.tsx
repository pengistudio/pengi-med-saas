import type React from "react";
import { useCallback } from "react";
import { useLocation } from "react-router";
import { cn } from "@/lib/utils";
import { useSidebarStore } from "@/store/sidebar-store";

type Props = {
	label: string;
	href: string;
	icon: React.ComponentType<{ className?: string }>;
};

const NavItem = ({ label, href, icon }: Props) => {
	const { isOpen: sidebarOpen } = useSidebarStore();
	const { close } = useSidebarStore();
	const location = useLocation();

	const isActive = location.pathname === href;

	const handleNavItemClick = useCallback(() => {
		if (window.innerWidth < 768) {
			close();
		}
	}, [close]);

	const Icon = icon;
	return (
		<a
			key={label}
			href={href}
			title={!sidebarOpen ? label : undefined}
			className={cn(
				"flex items-center rounded-lg px-3 py-2 transition-all duration-300 overflow-hidden",
				sidebarOpen ? "gap-3" : "gap-0",
				isActive
					? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
					: "text-sidebar-foreground hover:bg-sidebar-accent",
			)}
			onClick={handleNavItemClick}
		>
			<Icon className={cn("h-5 w-5 shrink-0", isActive && "text-primary")} />
			<span
				className={cn(
					"transition-all duration-300 truncate",
					sidebarOpen
						? "opacity-100 w-auto"
						: "opacity-0 w-0 overflow-hidden ml-0",
				)}
			>
				{label}
			</span>
		</a>
	);
};

export default NavItem;
