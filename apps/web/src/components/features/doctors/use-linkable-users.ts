import React from "react";
import type { Doctor } from "@/api/doctors-service";
import { getTeamMembers, type TeamMember } from "@/api/team-service";
import { useDoctors } from "@/store/doctors-store";

/** Team members of the tenant (fetched once per mount). */
export function useTeamMembers() {
	const [members, setMembers] = React.useState<TeamMember[]>([]);
	React.useEffect(() => {
		getTeamMembers().then((res) => {
			if (res.success && res.data) setMembers(res.data);
		});
	}, []);
	return members;
}

const memberLabel = (m: TeamMember) =>
	`${m.environment_name || m.user_name} (@${m.user_name})`;

/**
 * Team members that can be linked to `doctor` (or to a new doctor): those
 * without a profile, plus the doctor's own user. As select options.
 */
export function useLinkableUsers(doctor?: Doctor | null) {
	const members = useTeamMembers();
	const { doctors } = useDoctors();
	return React.useMemo(() => {
		const taken = new Set(
			doctors
				.filter((d) => d.user_id && d.ID !== doctor?.ID)
				.map((d) => d.user_id),
		);
		return members
			.filter((m) => !taken.has(m.user_id))
			.map((m) => ({ value: String(m.user_id), label: memberLabel(m) }));
	}, [members, doctors, doctor?.ID]);
}
