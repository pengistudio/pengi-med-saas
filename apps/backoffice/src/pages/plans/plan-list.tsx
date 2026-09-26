import { type Plan, plans } from "@/api/plan-service";
import { type ResourceColumn, ResourceList } from "@/lib/resource";

const columns: ResourceColumn<Plan>[] = [
	{
		header: "backoffice.plans.col.name",
		cell: (p) => p.name,
		className: "font-medium",
	},
	{
		header: "backoffice.plans.col.code",
		cell: (p) => p.code,
		className: "font-mono text-sm",
	},
	{
		header: "backoffice.plans.col.tier",
		cell: (p) => (
			<span className="inline-flex items-center justify-center w-7 h-7 rounded-full bg-muted text-xs font-bold">
				{p.tier ?? 1}
			</span>
		),
	},
	{
		header: "backoffice.plans.pricings.title",
		cell: (p) =>
			p.pricings && p.pricings.length > 0 ? (
				<div className="flex flex-wrap gap-1">
					{[...p.pricings]
						.sort((a, b) => a.months - b.months)
						.map((pr) => (
							<span
								key={pr.months}
								className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs font-mono"
							>
								{pr.months}m · ${pr.price.toFixed(0)}
							</span>
						))}
				</div>
			) : (
				<span className="text-sm text-muted-foreground">
					${p.price.toFixed(2)}/mes
				</span>
			),
	},
	{
		header: "backoffice.plans.col.features",
		cell: (p) => (
			<span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
				{p.Features?.length ?? 0}
			</span>
		),
	},
];

const PlanList = () => (
	<ResourceList resource={plans} columns={columns} itemLabel={(p) => p.name} />
);

export default PlanList;
