import { useText } from "@pengi/shared";
import { Link, useLocation } from "react-router";
import { cn } from "@/lib/utils";

const TABS = [
	{ href: "/billing", labelKey: "dashboard.billing.invoices" },
	{
		href: "/billing/credit-notes",
		labelKey: "dashboard.billing.credit-notes",
	},
	{ href: "/billing/debit-notes", labelKey: "dashboard.billing.debit-notes" },
] as const;

/** Route-link tab strip to switch between invoices, credit notes and debit notes. */
export function BillingDocumentsTabs() {
	const { textGet } = useText();
	const { pathname } = useLocation();
	const current = pathname.replace(/\/+$/, "");

	return (
		<nav aria-label={textGet("nav.item.documents")} className="min-w-0">
			<div className="inline-flex max-w-full items-center gap-1 overflow-x-auto rounded-lg bg-muted p-[3px] text-muted-foreground">
				{TABS.map((tab) => {
					const active = current === tab.href;
					return (
						<Link
							key={tab.href}
							to={tab.href}
							aria-current={active ? "page" : undefined}
							className={cn(
								"inline-flex h-7 shrink-0 items-center whitespace-nowrap rounded-md px-3 text-sm font-medium outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring",
								active
									? "bg-background text-foreground shadow-sm"
									: "hover:text-foreground",
							)}
						>
							{textGet(tab.labelKey)}
						</Link>
					);
				})}
			</div>
		</nav>
	);
}
