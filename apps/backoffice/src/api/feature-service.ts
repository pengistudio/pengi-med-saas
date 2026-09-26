import { resource } from "@/lib/resource/http-resource";
export interface Feature {
	ID: number;
	CreatedAt: string;
	UpdatedAt: string;
	code: string;
	name: string;
	permissions: {
		ID: string;
		name: string;
		category: string;
		description: string;
	}[];
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
