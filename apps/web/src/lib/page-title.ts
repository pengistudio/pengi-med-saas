import { navTitle } from "@pengi/ui";
import type { NavItemType } from "@/config/nav-config";

/** Pages reachable outside the sidebar (user menu, notification bell). */
const EXTRA_TITLES: { href: string; label: string }[] = [
	{ href: "/profile", label: "profile.title" },
	{ href: "/notifications", label: "notification.page.title" },
];

/** Header title for the current route: the matching nav item, or a page outside the sidebar. */
export function getPageTitle(
	navItems: NavItemType[],
	pathname: string,
	textGet: (key: string) => string,
): string | undefined {
	return navTitle(
		[
			...navItems,
			...EXTRA_TITLES.map(({ href, label }) => ({
				href,
				label: textGet(label),
			})),
		],
		pathname,
	);
}
