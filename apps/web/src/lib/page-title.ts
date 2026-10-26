import { navTitle } from "@pengi/ui";
import type { NavItemType } from "@/config/nav-config";

/** Pages reachable outside the sidebar (user menu, bell, settings hub, billing tabs). */
const EXTRA_TITLES: { href: string; label: string }[] = [
	{ href: "/billing/credit-notes", label: "dashboard.billing.credit-notes" },
	{ href: "/billing/debit-notes", label: "dashboard.billing.debit-notes" },
	{ href: "/billing/settings", label: "dashboard.billing.settings" },
	{ href: "/team", label: "team.title" },
	{ href: "/doctors", label: "doctors.title" },
	{ href: "/audit", label: "audit.title" },
	{ href: "/subscription", label: "subscription.nav.title" },
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
