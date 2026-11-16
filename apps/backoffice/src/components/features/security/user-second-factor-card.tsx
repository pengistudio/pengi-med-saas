import { useText } from "@pengi/shared";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@pengi/ui";
import React from "react";
import { type BackofficeUser, users } from "@/api/user-service";
import {
	SecondFactorBadges,
	SecondFactorResetActions,
} from "./admin-reset-actions";

/** Another admin's second-factor status, with the reset actions. */
export function UserSecondFactorCard({ id }: { id: string }) {
	const { textGet } = useText();
	const [user, setUser] = React.useState<BackofficeUser>();

	const load = React.useCallback(() => {
		users.get(id).then((res) => {
			if (res.success) setUser(res.data);
		});
	}, [id]);

	React.useEffect(() => {
		load();
	}, [load]);

	if (!user) return null;
	return (
		<Card>
			<CardHeader>
				<CardTitle>{textGet("backoffice.users.second_factor.title")}</CardTitle>
				<CardDescription>
					{textGet("backoffice.users.second_factor.description")}
				</CardDescription>
			</CardHeader>
			<CardContent className="flex flex-wrap items-center justify-between gap-3">
				<SecondFactorBadges user={user} />
				<div className="flex flex-wrap gap-2">
					<SecondFactorResetActions user={user} onChanged={load} />
				</div>
			</CardContent>
		</Card>
	);
}
