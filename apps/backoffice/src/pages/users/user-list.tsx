import { type BackofficeUser, users } from "@/api/user-service";
import { type ResourceColumn, ResourceList } from "@/lib/resource";

const columns: ResourceColumn<BackofficeUser>[] = [
	{
		header: "backoffice.users.col.name",
		cell: (u) => u.name,
		className: "font-medium",
	},
	{
		header: "backoffice.users.col.user_name",
		cell: (u) => u.user_name,
		className: "text-muted-foreground",
	},
];

const UserList = () => (
	<ResourceList
		resource={users}
		columns={columns}
		itemLabel={(u) => u.user_name}
	/>
);

export default UserList;
