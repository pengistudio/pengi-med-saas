import { type Feature, features } from "@/api/feature-service";
import { type ResourceColumn, ResourceList } from "@/lib/resource";

const columns: ResourceColumn<Feature>[] = [
	{
		header: "backoffice.features.col.code",
		cell: (f) => f.code,
		className: "font-mono text-sm",
	},
	{
		header: "backoffice.features.col.name",
		cell: (f) => f.name,
		className: "font-medium",
	},
	{
		header: "backoffice.features.col.permissions",
		cell: (f) => (
			<span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
				{f.permissions?.length ?? 0}
			</span>
		),
	},
];

const FeatureList = () => (
	<ResourceList
		resource={features}
		columns={columns}
		itemLabel={(f) => f.name}
	/>
);

export default FeatureList;
