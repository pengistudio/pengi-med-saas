import { resource } from "@/lib/resource/http-resource";
import type { Permission } from "./permission-service";
export interface Role {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	role: string;
	permissions: Permission[];
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
