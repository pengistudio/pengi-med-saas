import type { BaseModel } from "@pengi/shared";
import { resource } from "@/lib/resource/http-resource";
import type { Permission } from "./permission-service";
export interface Feature extends BaseModel {
	code: string;
	name: string;
	permissions: Permission[];
}

export interface CreateFeatureRequest extends Record<string, unknown> {
	code: string;
	name: string;
	permission_ids?: string[];
}

export interface UpdateFeatureRequest extends Record<string, unknown> {
	name?: string;
	permission_ids?: string[];
}

export const features = resource<
	Feature,
	CreateFeatureRequest,
	UpdateFeatureRequest
>("features");
