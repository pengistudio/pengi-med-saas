import { cn } from "@pengi/ui";
import { type Subscription, subscriptions } from "@/api/subscription-service";
import { type ResourceColumn, ResourceList } from "@/lib/resource";
import { formatExpiry } from "@/lib/subscription/term";

const statusColors: Record<string, string> = {
	active: "bg-emerald-500/10 text-emerald-600",
	expired: "bg-red-500/10 text-red-600",
	cancelled: "bg-zinc-500/10 text-zinc-500",
};

const companyName = (s: Subscription) =>
	s.company?.trade_name ?? `#${s.CompanyID}`;

const columns: ResourceColumn<Subscription>[] = [
	{ header: "backoffice.subscriptions.col.company", cell: companyName },
	{
		header: "backoffice.subscriptions.col.plan",
		cell: (s) => s.plan?.name ?? s.plan_code,
		className: "font-medium",
	},
	{
		header: "backoffice.subscriptions.col.status",
		cell: (s) => (
			<span
				className={cn(
					"inline-flex items-center rounded-full px-2 py-1 text-xs font-medium",
					statusColors[s.status] ?? "bg-muted text-muted-foreground",
				)}
			>
				{s.status}
			</span>
		),
	},
	{
		header: "backoffice.subscriptions.col.expires",
		cell: (s) => formatExpiry(s.expires_at),
		className: "text-muted-foreground",
	},
];

const SubscriptionList = () => (
	<ResourceList
		resource={subscriptions}
		columns={columns}
		itemLabel={(s) => `${companyName(s)} · ${s.plan?.name ?? s.plan_code}`}
	/>
);

export default SubscriptionList;
