import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from ".";

const signatureService = createHttpService(apiWithTenant);

/** Signature fields every signable document carries. */
export interface DocumentSignature {
	signed_by_id?: number | null;
	signed_at?: string | null;
	signer_name?: string;
}

/** The current user's electronic signature (P12) in this tenant. */
export interface MySignature {
	configured: boolean;
	subject_name?: string;
	subject_serial?: string;
	issuer?: string;
	not_after?: string;
	expired: boolean;
	expiring_soon: boolean;
}

export const getMySignature = async (): Promise<ServiceResponse<MySignature>> =>
	signatureService.get<MySignature>("/signatures/me");

export const uploadMySignature = async (
	file: File,
	password: string,
): Promise<ServiceResponse<MySignature>> => {
	const formData = new FormData();
	formData.append("signature", file);
	formData.append("password", password);
	return signatureService.putForm<MySignature>("/signatures/me", formData, {
		notifySuccess: true,
		notifyError: true,
	});
};

export const deleteMySignature = async (): Promise<
	ServiceResponse<MySignature>
> =>
	signatureService.delete<MySignature>("/signatures/me", {
		notifySuccess: true,
		notifyError: true,
	});

export const signMedicalReport = async (
	reportId: number,
): Promise<ServiceResponse<DocumentSignature>> =>
	signatureService.post<DocumentSignature>(
		`/clinical/reports/${reportId}/sign`,
		undefined,
		{ notifySuccess: true, notifyError: true },
	);

export const signMedicalCertificate = async (
	certificateId: number,
): Promise<ServiceResponse<DocumentSignature>> =>
	signatureService.post<DocumentSignature>(
		`/clinical/certificates/${certificateId}/sign`,
		undefined,
		{ notifySuccess: true, notifyError: true },
	);

export const signPrescription = async (
	medicalRecordId: number,
): Promise<ServiceResponse<DocumentSignature>> =>
	signatureService.post<DocumentSignature>(
		`/clinical/records/${medicalRecordId}/prescription/sign`,
		undefined,
		{ notifySuccess: true, notifyError: true },
	);
