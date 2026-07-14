import { apiWithTenant } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(apiWithTenant);

export interface TeamMember {
	environment_id: number;
	user_id: number;
	user_name: string;
	email: string;
	role_id: number;
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

export const updateTeamMemberRole = (
	environmentId: number,
	roleId: number,
): Promise<ServiceResponse<{ environment_id: number; role_id: number }>> =>
	httpService.put<{ environment_id: number; role_id: number }>(
		`/companies/team/${environmentId}/role`,
		{ role_id: roleId },
		{ notifyError: true, notifySuccess: true },
	);
