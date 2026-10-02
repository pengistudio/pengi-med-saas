import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from ".";

const documentTemplateService = createHttpService(apiWithTenant);

/** A printable document whose template the tenant can replace. */
export type DocumentTemplateId =
	| "prescription"
	| "medical_report"
	| "medical_certificate"
	| "invoice_ride";

export interface DocumentTemplate {
	id: DocumentTemplateId;
	has_custom: boolean;
	paper: "a4_portrait" | "a5_landscape" | "custom";
}

/** The printable documents of the tenant's plan modules. */
export const getDocumentTemplates = async (): Promise<
	ServiceResponse<DocumentTemplate[]>
> =>
	documentTemplateService.get<DocumentTemplate[]>("/document-templates", {
		notifyError: true,
	});

export const downloadDefaultDocumentTemplate = async (
	id: DocumentTemplateId,
): Promise<ServiceResponse<Blob>> =>
	documentTemplateService.get<Blob>(`/document-templates/${id}/default`, {
		responseType: "blob",
		notifyError: true,
	});

export const downloadCustomDocumentTemplate = async (
	id: DocumentTemplateId,
): Promise<ServiceResponse<Blob>> =>
	documentTemplateService.get<Blob>(`/document-templates/${id}/custom`, {
		responseType: "blob",
		notifyError: true,
	});

/** Validates and stores the tenant's template; the error toast says which rule failed. */
export const uploadDocumentTemplate = async (
	id: DocumentTemplateId,
	file: File,
): Promise<ServiceResponse<DocumentTemplate>> => {
	const form = new FormData();
	form.append("template", file);
	return documentTemplateService.putForm<DocumentTemplate>(
		`/document-templates/${id}`,
		form,
		{ notifySuccess: true, notifyError: true },
	);
};

/** Removes the tenant's template so the default applies again. */
export const deleteDocumentTemplate = async (
	id: DocumentTemplateId,
): Promise<ServiceResponse<DocumentTemplate>> =>
	documentTemplateService.delete<DocumentTemplate>(
		`/document-templates/${id}`,
		{ notifySuccess: true, notifyError: true },
	);

/** PDF of the document's sample data with the tenant's effective template. */
export const previewDocumentTemplate = async (
	id: DocumentTemplateId,
): Promise<ServiceResponse<Blob>> =>
	documentTemplateService.post<Blob>(
		`/document-templates/${id}/preview`,
		undefined,
		{ responseType: "blob", notifyError: true },
	);
