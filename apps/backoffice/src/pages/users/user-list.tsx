import { type BackofficeUser, users } from "@/api/user-service";
import {
	SecondFactorBadges,
	SecondFactorResetActions,
} from "@/components/features/security/admin-reset-actions";
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
	{
		header: "backoffice.users.col.second_factor",
		cell: (u) => <SecondFactorBadges user={u} />,
	},
];

const UserList = () => (
	<ResourceList
		resource={users}
		columns={columns}
		itemLabel={(u) => u.user_name}
		rowActions={(u, refetch) => (
			<SecondFactorResetActions user={u} onChanged={refetch} />
		)}
	/>
);

export default UserList;
