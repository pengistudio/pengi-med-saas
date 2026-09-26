import { type Role, roles } from "@/api/role-service";
import { type ResourceColumn, ResourceList } from "@/lib/resource";

const columns: ResourceColumn<Role>[] = [
	{
		header: "backoffice.roles.col.name",
		cell: (r) => r.role,
		className: "font-medium",
	},
	{
		header: "backoffice.features.col.permissions",
		cell: (r) => (
			<span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
				{r.permissions?.length ?? 0}
			</span>
		),
	},
];

const RoleList = () => (
	<ResourceList resource={roles} columns={columns} itemLabel={(r) => r.role} />
);

export default RoleList;
