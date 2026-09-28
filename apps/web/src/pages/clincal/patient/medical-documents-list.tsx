import { useText } from "@pengi/shared";
import {
	Badge,
	Button,
	Card,
	CardContent,
	CardHeader,
	CardTitle,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
	Text,
} from "@pengi/ui";
import { ArrowLeft, FileCheck, FileText, Loader2, Printer } from "lucide-react";
import React from "react";
import { useNavigate, useSearchParams } from "react-router";
import {
	downloadMedicalCertificatePdf,
	downloadMedicalReportPdf,
	emailMedicalCertificate,
	emailMedicalReport,
	getMedicalCertificates,
	getMedicalReports,
	getPatientById,
	type MedicalCertificate,
	type MedicalReport,
	type Patient,
} from "@/api/clinical-service";
import {
	type DocumentSignature,
	signMedicalCertificate,
	signMedicalReport,
} from "@/api/signature-service";
import { SendEmailPopover } from "@/components/custom/send-email-popover";
import { SignDocumentButton } from "@/components/custom/sign-document-button";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { DashboardLayout } from "@/sections/template/dashboard-template";

type DocumentType = "report" | "certificate";

interface DocumentRow {
	id: number;
	type: DocumentType;
	createdAt: string;
	signature: DocumentSignature;
}

export default function MedicalDocumentsListPage() {
	const { formatDateTime } = useText();
	const navigate = useNavigate();
	const { checkPermission } = usePermission();
	const [searchParams] = useSearchParams();
	const patientId = Number(searchParams.get("patient_id"));

	const [patient, setPatient] = React.useState<Patient | null>(null);
	const [loading, setLoading] = React.useState(true);
	const [rows, setRows] = React.useState<DocumentRow[]>([]);
	const [printingId, setPrintingId] = React.useState<string | null>(null);

	// Show the spinner again when the patient changes (adjust state during render).
	const [prevPatientId, setPrevPatientId] = React.useState(patientId);
	if (patientId !== prevPatientId) {
		setPrevPatientId(patientId);
		if (patientId) setLoading(true);
	}

	const loadDocuments = React.useCallback(() => {
		if (!patientId) return;
		Promise.all([
			getPatientById(patientId),
			getMedicalReports(patientId),
			getMedicalCertificates(patientId),
		]).then(([patientRes, reportsRes, certificatesRes]) => {
			if (patientRes.success && patientRes.data) {
				setPatient(patientRes.data);
			}
			const reports: MedicalReport[] = reportsRes.success
				? (reportsRes.data ?? [])
				: [];
			const certificates: MedicalCertificate[] = certificatesRes.success
				? (certificatesRes.data ?? [])
				: [];
			const merged: DocumentRow[] = [
				...reports.map((r) => ({
					id: r.ID,
					type: "report" as const,
					createdAt: r.CreatedAt,
					signature: r,
				})),
				...certificates.map((c) => ({
					id: c.ID,
					type: "certificate" as const,
					createdAt: c.CreatedAt,
					signature: c,
				})),
			].sort(
				(a, b) =>
					new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
			);
			setRows(merged);
			setLoading(false);
		});
	}, [patientId]);

	React.useEffect(() => {
		loadDocuments();
	}, [loadDocuments]);

	async function handlePrint(row: DocumentRow) {
		const key = `${row.type}-${row.id}`;
		setPrintingId(key);
		const res =
			row.type === "report"
				? await downloadMedicalReportPdf(row.id)
				: await downloadMedicalCertificatePdf(row.id);
		if (res.success && res.data) {
			const url = window.URL.createObjectURL(res.data);
			window.open(url, "_blank");
		}
		setPrintingId(null);
	}

	async function handleSign(row: DocumentRow) {
		const res =
			row.type === "report"
				? await signMedicalReport(row.id)
				: await signMedicalCertificate(row.id);
		if (res.success && res.data) {
			const signature = res.data;
			setRows((current) =>
				current.map((r) =>
					r.type === row.type && r.id === row.id ? { ...r, signature } : r,
				),
			);
		}
		return res.success;
	}

	async function handleSendEmail(row: DocumentRow, email: string) {
		if (row.type === "report") {
			await emailMedicalReport(row.id, email);
		} else {
			await emailMedicalCertificate(row.id, email);
		}
	}

	const fullName = patient
		? patient.full_name || `${patient.last_name} ${patient.first_name}`
		: "";

	return (
		<DashboardLayout>
			<main className="grid items-start gap-4">
				<Card className="max-w-4xl mx-auto w-full">
					<CardHeader>
						<Button
							type="button"
							variant="ghost"
							size="sm"
							className="w-fit -ml-2 mb-2"
							onClick={() => navigate("/clinical")}
						>
							<ArrowLeft className="mr-2 h-4 w-4" />
							<Text uuid="clinical.medical_documents.back" />
						</Button>
						<CardTitle>
							<Text uuid="clinical.medical_documents.title" />
							{fullName ? ` — ${fullName}` : ""}
						</CardTitle>
						<p className="text-sm text-muted-foreground">
							<Text uuid="clinical.medical_documents.description" />
						</p>
						<div className="flex flex-wrap gap-2 pt-2">
							{checkPermission([
								PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_REPORT,
							]) && (
								<Button
									type="button"
									variant="outline"
									size="sm"
									onClick={() =>
										navigate(
											`/clinical/medical-reports/create?patient_id=${patientId}`,
										)
									}
								>
									<FileText className="mr-2 h-4 w-4" />
									<Text uuid="clinical.medical_documents.new_report" />
								</Button>
							)}
							{checkPermission([
								PERMISSIONS.MEDICAL_RECORD
									.PERMISSION_CREATE_MEDICAL_CERTIFICATE,
							]) && (
								<Button
									type="button"
									variant="outline"
									size="sm"
									onClick={() =>
										navigate(
											`/clinical/medical-certificates/create?patient_id=${patientId}`,
										)
									}
								>
									<FileCheck className="mr-2 h-4 w-4" />
									<Text uuid="clinical.medical_documents.new_certificate" />
								</Button>
							)}
						</div>
					</CardHeader>
					<CardContent>
						{loading ? (
							<div className="flex justify-center py-8">
								<Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
							</div>
						) : rows.length === 0 ? (
							<p className="text-sm text-muted-foreground text-center py-8">
								<Text uuid="clinical.medical_documents.empty" />
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>
											<Text uuid="clinical.medical_documents.type" />
										</TableHead>
										<TableHead>
											<Text uuid="clinical.medical_documents.created_at" />
										</TableHead>
										<TableHead className="text-right">
											<Text uuid="table.actions" />
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{rows.map((row) => {
										const key = `${row.type}-${row.id}`;
										return (
											<TableRow key={key}>
												<TableCell>
													<Badge variant="secondary">
														{row.type === "report" ? (
															<Text uuid="clinical.medical_documents.type.report" />
														) : (
															<Text uuid="clinical.medical_documents.type.certificate" />
														)}
													</Badge>
												</TableCell>
												<TableCell>{formatDateTime(row.createdAt)}</TableCell>
												<TableCell className="text-right">
													<div className="flex items-center justify-end gap-1">
														<SignDocumentButton
															compact
															signature={row.signature}
															onSign={() => handleSign(row)}
														/>
														<Button
															type="button"
															variant="ghost"
															size="icon"
															disabled={printingId === key}
															onClick={() => handlePrint(row)}
														>
															{printingId === key ? (
																<Loader2 className="h-4 w-4 animate-spin" />
															) : (
																<Printer className="h-4 w-4" />
															)}
														</Button>
														<SendEmailPopover
															defaultEmail={patient?.email ?? ""}
															onSend={(email) => handleSendEmail(row, email)}
														/>
													</div>
												</TableCell>
											</TableRow>
										);
									})}
								</TableBody>
							</Table>
						)}
					</CardContent>
				</Card>
			</main>
		</DashboardLayout>
	);
}
