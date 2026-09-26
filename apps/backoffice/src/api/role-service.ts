import { resource } from "@/lib/resource/http-resource";
export interface Role {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	role: string;
	permissions: {
		ID: string;
		name: string;
		category: string;
		description: string;
	}[];
}

export interface CreateRoleRequest extends Record<string, unknown> {
	role: string;
	permission_ids?: string[];
}

export interface UpdateRoleRequest extends Record<string, unknown> {
	role?: string;
	permission_ids?: string[];
}

export const roles = resource<Role, CreateRoleRequest, UpdateRoleRequest>(
	"roles",
);
