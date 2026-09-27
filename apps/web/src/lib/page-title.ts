import type { NavItemType } from "@/config/nav-config";

/** Pages reachable outside the sidebar (user menu, notification bell). */
const EXTRA_TITLES: { href: string; label: string }[] = [
	{ href: "/profile", label: "profile.title" },
	{ href: "/notifications", label: "notification.page.title" },
];

function matches(href: string, pathname: string): boolean {
	if (href === "/") return pathname === "/";
	return pathname === href || pathname.startsWith(`${href}/`);
}

/**
 * Header title for the current route: the nav item (or nested item) whose URL
 * is the longest prefix of `pathname`, so `/clinical/medical-records/12`
 * reads as "Pacientes" and `/billing/settings` beats `/billing`.
 */
export function getPageTitle(
	navItems: NavItemType[],
	pathname: string,
	textGet: (key: string) => string,
): string | undefined {
	const candidates = [
		...navItems.flatMap((item) => [
			...(item.href ? [{ href: item.href, label: item.label }] : []),
			...(item.accordionItems ?? []),
		]),
		...EXTRA_TITLES.map(({ href, label }) => ({ href, label: textGet(label) })),
	];

	let best: { href: string; label: string } | undefined;
	for (const candidate of candidates) {
		if (
			matches(candidate.href, pathname) &&
			(!best || candidate.href.length > best.href.length)
		) {
			best = candidate;
		}
	}
	return best?.label;
}
