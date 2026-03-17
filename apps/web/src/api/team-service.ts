import { apiWithTenant } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(apiWithTenant);

export interface TeamMember {
	environment_id: number;
	user_id: number;
	user_name: string;
	email: string;
	role_name: string;
	environment_name: string;
}

export interface TeamRole {
	ID: number;
	role: string;
}

export const getTeamMembers = (): Promise<ServiceResponse<TeamMember[]>> =>
	httpService.get<TeamMember[]>("/companies/team", { notifyError: true });

export const getTeamRoles = (): Promise<ServiceResponse<TeamRole[]>> =>
	httpService.get<TeamRole[]>("/companies/team/roles", { notifyError: true });

export const generateInviteLink = (
	roleId: number,
): Promise<ServiceResponse<{ token: string }>> =>
	httpService.post<{ token: string }>(
		"/companies/team/invite-link",
		{ role_id: roleId },
		{ notifyError: true },
	);
