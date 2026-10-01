type TitledLink = { label: string; href: string };
type TitledEntry = {
	label: string;
	href?: string;
	accordionItems?: TitledLink[];
};

function matches(href: string, pathname: string): boolean {
	if (href === "/") return pathname === "/";
	return pathname === href || pathname.startsWith(`${href}/`);
}

/**
 * The page title for `pathname`: the label of the nav link (or nested link)
 * whose href is its longest prefix, so `/clinical/medical-records/12` reads as
 * "Pacientes" and `/billing/settings` beats `/billing`. `undefined` when no
 * link matches, so the caller picks the fallback.
 */
export function navTitle(
	entries: TitledEntry[],
	pathname: string,
): string | undefined {
	const links = entries.flatMap((entry) => [
		...(entry.href ? [{ href: entry.href, label: entry.label }] : []),
		...(entry.accordionItems ?? []),
	]);

	let best: TitledLink | undefined;
	for (const link of links) {
		if (
			matches(link.href, pathname) &&
			(!best || link.href.length > best.href.length)
		) {
			best = link;
		}
	}
	return best?.label;
}
