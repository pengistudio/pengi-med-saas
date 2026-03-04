import { api } from ".";
import { createHttpService, type ServiceResponse } from "./fetch";

const httpService = createHttpService(api);

export interface Feature {
	ID: number;
	createdAt: string;
	updatedAt: string;
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

export const getFeatures = (): Promise<ServiceResponse<Feature[]>> =>
	httpService.get<Feature[]>("/backoffice/features");

export const getFeatureByID = (
	id: number | string,
): Promise<ServiceResponse<Feature>> =>
	httpService.get<Feature>(`/backoffice/features/${id}`);

export const createFeature = (
	data: CreateFeatureRequest,
): Promise<ServiceResponse<Feature>> =>
	httpService.post<Feature>("/backoffice/features", data, {
		notifySuccess: true,
		notifyError: true,
	});

export const updateFeature = (
	id: number | string,
	data: UpdateFeatureRequest,
): Promise<ServiceResponse<Feature>> =>
	httpService.put<Feature>(`/backoffice/features/${id}`, data, {
		notifySuccess: true,
		notifyError: true,
	});

export const deleteFeature = (
	id: number | string,
): Promise<ServiceResponse<null>> =>
	httpService.delete<null>(`/backoffice/features/${id}`, {
		notifySuccess: true,
		notifyError: true,
	});
