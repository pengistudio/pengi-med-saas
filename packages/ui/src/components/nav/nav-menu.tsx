import type React from "react";
import { useViewport } from "../../hooks/use-viewport";
import { cn } from "../../lib/utils";
import { useAppShell } from "../app-shell/app-shell-context";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "../dropdown-menu";

type Icon = React.ComponentType<{ className?: string }>;

export type NavMenuItem = {
	label: string;
	/** Second line under the label. */
	description?: string;
	icon: Icon;
	/** External link (https:, mailto:), opened in a new tab. */
	href: string;
};

type Props = {
	label: string;
	icon: Icon;
	menuItems: NavMenuItem[];
};

/**
 * A sidebar entry that opens a menu of external links (e.g. help: WhatsApp,
 * email) instead of navigating. Looks like a NavItem; the menu opens beside
 * the rail on desktop and above the entry in the phone drawer.
 */
const NavMenu = ({ label, icon: Icon, menuItems }: Props) => {
	const { expanded } = useAppShell();
	const { isPhone } = useViewport();
	// The wrapper keeps the menu's focus guards and portal (siblings of the
	// trigger) out of the sidebar's spaced list, which would shift it on open.
	return (
		<div>
			<DropdownMenu>
				<DropdownMenuTrigger
					title={!expanded ? label : undefined}
					className={cn(
						"relative flex min-h-10 w-full items-center rounded-lg px-3 py-2 text-sidebar-foreground outline-none transition-all duration-300 hover:bg-sidebar-accent focus-visible:ring-3 focus-visible:ring-ring/50 data-popup-open:bg-sidebar-accent md:min-h-9 md:py-1.5 md:text-sm motion-reduce:transition-none",
						expanded ? "gap-3" : "gap-0",
					)}
				>
					<Icon className="h-5 w-5 shrink-0" />
					<span
						className={cn(
							"truncate text-left transition-all duration-300 motion-reduce:transition-none",
							expanded
								? "w-auto opacity-100"
								: "ml-0 w-0 overflow-hidden opacity-0",
						)}
					>
						{label}
					</span>
				</DropdownMenuTrigger>
				<DropdownMenuContent
					side={isPhone ? "top" : "right"}
					align="end"
					sideOffset={8}
					className="w-max max-w-[calc(100vw-1.5rem)] min-w-64 p-1.5"
				>
					{menuItems.map((item) => {
						const ItemIcon = item.icon;
						return (
							<DropdownMenuItem
								key={item.href}
								// Base UI renders the item's children inside the anchor.
								render={
									<a
										href={item.href}
										target="_blank"
										rel="noopener noreferrer"
									/>
								}
								className="min-h-11 items-start gap-3 px-2.5 py-2"
							>
								<span className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-full bg-primary/12 text-primary">
									<ItemIcon className="size-4" />
								</span>
								<span className="flex min-w-0 flex-col">
									<span className="font-medium">{item.label}</span>
									{item.description && (
										<span className="text-muted-foreground text-xs">
											{item.description}
										</span>
									)}
								</span>
							</DropdownMenuItem>
						);
					})}
				</DropdownMenuContent>
			</DropdownMenu>
		</div>
	);
};

export default NavMenu;
