import { SelectLanguage, useText } from "@pengi/shared";
import {
	AppShell,
	Avatar,
	AvatarFallback,
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
	navTitle,
} from "@pengi/ui";
import { HelpCircle, Power, Shield } from "lucide-react";
import { Suspense, useMemo } from "react";
import { Outlet, useLocation } from "react-router";
import { createNavItems } from "@/config/nav-config";
import { useSession } from "@/lib/session";

/** The frame of every signed-in page; the route tree renders pages in its Outlet. */
export function DashboardLayout() {
	const { logout } = useSession();
	const { textGet } = useText();

	const { pathname } = useLocation();
	const navItems = useMemo(() => createNavItems(textGet), [textGet]);
	const footerNavItems = [
		{
			label: textGet("backoffice.help"),
			href: "/",
			icon: HelpCircle,
			matchActive: false,
		},
	];

	return (
		<AppShell
			brand={{ icon: Shield, name: "Pengi Admin" }}
			nav={navItems}
			footerNav={footerNavItems}
			title={
				navTitle(navItems, pathname) ?? textGet("backoffice.nav.dashboard")
			}
			actions={
				<>
					<SelectLanguage />
					<DropdownMenu>
						<DropdownMenuTrigger className="relative h-10 w-10 cursor-pointer rounded-full focus-visible:outline-none">
							<Avatar className="h-10 w-10">
								<AvatarFallback>BO</AvatarFallback>
							</Avatar>
						</DropdownMenuTrigger>
						<DropdownMenuContent align="end" className="w-56">
							<DropdownMenuGroup>
								<DropdownMenuLabel>
									{textGet("backoffice.dropdown.account")}
								</DropdownMenuLabel>
								<DropdownMenuSeparator />
								<DropdownMenuItem>
									{textGet("backoffice.dropdown.settings")}
								</DropdownMenuItem>
							</DropdownMenuGroup>
							<DropdownMenuSeparator />
							{/* RequireSession sends the now-anonymous user to /login. */}
							<DropdownMenuItem onClick={() => logout()} variant="destructive">
								<Power className="w-4 h-4 text-red-500" />
								{textGet("backoffice.dropdown.logout")}
							</DropdownMenuItem>
						</DropdownMenuContent>
					</DropdownMenu>
				</>
			}
		>
			{/* Lazy pages load inside the shell, so the frame stays while they do. */}
			<Suspense fallback={null}>
				<Outlet />
			</Suspense>
		</AppShell>
	);
}
