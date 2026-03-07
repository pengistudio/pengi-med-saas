import {
	Building2,
	HelpCircle,
	Menu,
	PanelLeft,
	PanelLeftClose,
	Power,
} from "lucide-react";
import type React from "react";
import { memo, useCallback, useMemo } from "react";
import NavAccordion from "@/components/custom/nav/nav-accordion";
import NavItem from "@/components/custom/nav/nav-item";
import SelectLanguage from "@/components/custom/select-language";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import useAuth from "@/hooks/use-auth";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";
import { useSessionStore } from "@/store/session-store";
import { useSidebarStore } from "@/store/sidebar-store";

interface DashboardLayoutProps {
	children: React.ReactNode;
}

import { createNavItems } from "@/config/nav-config";

function DashboardLayoutComponent({ children }: DashboardLayoutProps) {
	const { logout } = useAuth();
	const { textGet } = useText();
	const { isOpen: sidebarOpen, toggle, close, open } = useSidebarStore();

	const { environment } = useSessionStore();

	const handleAvatarFallbackText = useCallback(() => {
		return environment?.name
			? environment.name
					.split(" ")
					.map((n) => n[0])
					.join("")
			: "";
	}, [environment?.name]);

	// Use useMemo with stable reference
	const navItems = useMemo(() => createNavItems(textGet), [textGet]);

	const handleLogout = useCallback(() => {
		logout();
	}, [logout]);

	return (
		<div className="flex h-screen bg-background overflow-hidden max-h-screen">
			{/* Sidebar */}
			<aside
				className={cn(
					"flex flex-col border-r border-border bg-sidebar transition-all duration-500 ease-in-out shrink-0",
					sidebarOpen ? "w-64" : "w-16",
					"max-md:fixed max-md:inset-y-0 max-md:left-0 max-md:z-50 max-md:w-64",
					!sidebarOpen && "max-md:-translate-x-full",
				)}
			>
				{/* Sidebar Header */}
				<div className="flex h-16 items-center justify-center border-b border-sidebar-border px-4 overflow-hidden">
					{sidebarOpen ? (
						<div className="flex flex-1 items-center justify-between min-w-0">
							<div className="flex items-center gap-2 min-w-0 flex-1">
								<div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary shadow-lg transition-all duration-300 hover:shadow-xl">
									<Building2
										className="h-5 w-5 text-primary-foreground"
										strokeWidth={1.75}
									/>
								</div>
								<span className="text-lg font-semibold text-sidebar-foreground truncate min-w-0">
									{environment?.trade_name}
								</span>
							</div>
							<Button
								variant="ghost"
								size="icon"
								onClick={close}
								className="h-8 w-8 shrink-0 text-sidebar-foreground hover:bg-sidebar-accent transition-colors duration-300 max-md:hidden"
							>
								<PanelLeftClose className="h-5 w-5" />
							</Button>
						</div>
					) : (
						<Button
							variant="ghost"
							size="icon"
							onClick={open}
							className="h-10 w-10 shrink-0 text-sidebar-foreground hover:bg-sidebar-accent transition-colors duration-300 max-md:hidden"
						>
							<PanelLeft className="h-6 w-6" />
						</Button>
					)}
				</div>

				{/* Navigation */}
				<nav className="flex-1 space-y-1 p-2 overflow-hidden">
					{navItems.map((items) => {
						if (items.accordionItems) {
							return <NavAccordion key={items.label} {...items} />;
						}
						return <NavItem key={items.label} {...items} />;
					})}
				</nav>

				{/* Sidebar Footer */}
				<div className="border-t border-sidebar-border p-2 py-4 overflow-hidden">
					<a
						href="/"
						title={!sidebarOpen ? textGet("dashboard.help") : undefined}
						className={cn(
							"flex items-center rounded-lg px-3 py-2 text-sidebar-foreground transition-all duration-300 hover:bg-sidebar-accent overflow-hidden",
							sidebarOpen ? "gap-3" : "gap-0",
						)}
					>
						<HelpCircle className="h-5 w-5 shrink-0" />
						<span
							className={cn(
								"transition-all duration-300 truncate",
								sidebarOpen
									? "opacity-100 w-auto"
									: "opacity-0 w-0 overflow-hidden ml-0",
							)}
						>
							{textGet("dashboard.help")}
						</span>
					</a>
				</div>
			</aside>

			{sidebarOpen && (
				// biome-ignore lint/a11y/noStaticElementInteractions: here its fine
				// biome-ignore lint/a11y/useKeyWithClickEvents: here its fine
				<div
					className="fixed inset-0 z-40 bg-black/50 md:hidden backdrop-blur-sm"
					onClick={close}
				/>
			)}

			{/* Main Content */}
			<div className="flex flex-1 flex-col min-w-0 overflow-hidden">
				{/* Navbar */}
				<header className="flex h-16 items-center justify-between border-b border-border bg-card px-4 md:px-6 shrink-0">
					<div className="flex items-center gap-4">
						<Button
							variant="ghost"
							size="icon"
							className="md:hidden"
							onClick={toggle}
						>
							<Menu className="h-5 w-5" />
						</Button>
						<span className="text-lg font-semibold">
							{textGet("dashboard.title")}
						</span>
					</div>

					<div className="flex items-center gap-2">
						<SelectLanguage />
						<DropdownMenu>
							<DropdownMenuTrigger>
								<div className="relative h-10 w-10 rounded-full cursor-pointer">
									<Avatar className="h-10 w-10">
										<AvatarImage
											src="/placeholder.svg?height=40&width=40"
											alt="User"
										/>
										<AvatarFallback>
											{handleAvatarFallbackText()}
										</AvatarFallback>
									</Avatar>
								</div>
							</DropdownMenuTrigger>
							<DropdownMenuContent align="end" className="w-56">
								<DropdownMenuGroup>
									<DropdownMenuLabel>
										{textGet("dashboard.dropdown.profile.title")}
									</DropdownMenuLabel>
									<DropdownMenuSeparator />
									<DropdownMenuItem
										onClick={() => (window.location.href = "/profile")}
									>
										{textGet("dashboard.dropdown.profile")}
									</DropdownMenuItem>
									<DropdownMenuItem>
										{textGet("dashboard.dropdown.settings")}
									</DropdownMenuItem>
								</DropdownMenuGroup>
								<DropdownMenuSeparator />
								<DropdownMenuItem onClick={handleLogout} variant="destructive">
									<Power className="w-4 h-4 text-red-500" />
									{textGet("dashboard.dropdown.logout")}
								</DropdownMenuItem>
							</DropdownMenuContent>
						</DropdownMenu>
					</div>
				</header>

				{/* Page Content */}
				<main className="flex-1 overflow-auto p-4 md:p-6 relative">
					{children}
				</main>
			</div>
		</div>
	);
}

export const DashboardLayout = memo(DashboardLayoutComponent);
