import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardHeader,
	CardTitle,
	FormInput,
	FormTextArea,
	Input,
	Text,
} from "@pengi/ui";
import { ArrowLeft, Loader2, Mail, Printer, Save } from "lucide-react";
import React from "react";
import { useNavigate, useSearchParams } from "react-router";
import { z } from "zod";
import {
	createMedicalCertificate,
	downloadMedicalCertificatePdf,
	emailMedicalCertificate,
	getPatientById,
	type Patient,
} from "@/api/clinical-service";
import { Form } from "@/components/forms/form";
import { FormCalendar } from "@/components/forms/form-calendar";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const certificateSchema = z.object({
	diagnosis: z.string({ error: "Campo requerido" }).min(1, "Campo requerido"),
	observations: z.string().optional(),
	rest_days: z.coerce.number().optional(),
	rest_from: z.date().optional(),
	rest_to: z.date().optional(),
});

export default function GenerateMedicalCertificatePage() {
	const navigate = useNavigate();
	const { textGet } = useText();
	const [searchParams] = useSearchParams();
	const patientId = Number(searchParams.get("patient_id"));

	const [patient, setPatient] = React.useState<Patient | null>(null);
	const [loadingPatient, setLoadingPatient] = React.useState(true);
	const [saving, setSaving] = React.useState(false);
	const [printing, setPrinting] = React.useState(false);
	const [sending, setSending] = React.useState(false);
	const [savedCertificateId, setSavedCertificateId] = React.useState<
		number | null
	>(null);
	const [showEmailInput, setShowEmailInput] = React.useState(false);
	const [email, setEmail] = React.useState("");

	React.useEffect(() => {
		if (!patientId) return;
		getPatientById(patientId).then((res) => {
			if (res.success && res.data) {
				setPatient(res.data);
				setEmail(res.data.email ?? "");
			}
			setLoadingPatient(false);
		});
	}, [patientId]);

	async function onSubmit(values: z.infer<typeof certificateSchema>) {
		setSaving(true);
		const res = await createMedicalCertificate(patientId, {
			diagnosis: values.diagnosis,
			observations: values.observations ?? "",
			rest_days: values.rest_days ?? null,
			rest_from: values.rest_from ? values.rest_from.toISOString() : null,
			rest_to: values.rest_to ? values.rest_to.toISOString() : null,
		});
		if (res.success && res.data) {
			setSavedCertificateId(res.data.ID);
		}
		setSaving(false);
	}

	async function handlePrint() {
		if (!savedCertificateId) return;
		setPrinting(true);
		const res = await downloadMedicalCertificatePdf(savedCertificateId);
		if (res.success && res.data) {
			const url = window.URL.createObjectURL(res.data);
			window.open(url, "_blank");
		}
		setPrinting(false);
	}

	async function handleSendEmail() {
		if (!savedCertificateId || !email) return;
		setSending(true);
		await emailMedicalCertificate(savedCertificateId, email);
		setSending(false);
	}

	const fullName = patient
		? patient.full_name || `${patient.last_name} ${patient.first_name}`
		: "";

	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<Card className="max-w-3xl mx-auto w-full">
					<CardHeader>
						<Button
							type="button"
							variant="ghost"
							size="sm"
							className="w-fit -ml-2 mb-2"
							onClick={() =>
								navigate(`/clinical/medical-documents?patient_id=${patientId}`)
							}
						>
							<ArrowLeft className="mr-2 h-4 w-4" />
							<Text uuid="clinical.medical_documents.back" />
						</Button>
						<CardTitle>
							<Text uuid="dialog.medical_certificate.title" />
						</CardTitle>
						<p className="text-sm text-muted-foreground">
							<Text uuid="dialog.medical_certificate.description" />
						</p>
					</CardHeader>
					<CardContent className="space-y-4">
						{loadingPatient || !patient ? (
							<div className="flex justify-center py-8">
								<Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
							</div>
						) : (
							<>
								<div className="rounded-md bg-muted p-3 grid grid-cols-2 gap-2 text-sm">
									<div>
										<span className="text-muted-foreground">
											<Text uuid="clinical.patient.name" />:
										</span>{" "}
										{fullName}
									</div>
									<div>
										<span className="text-muted-foreground">
											<Text uuid="clinical.patient.document" />:
										</span>{" "}
										{patient.document}
									</div>
									<div className="col-span-2">
										<span className="text-muted-foreground">
											<Text uuid="dialog.medical_report.created_at" />:
										</span>{" "}
										{new Date().toLocaleDateString("es-EC")}
									</div>
								</div>

								<Form
									schema={certificateSchema}
									defaultValues={{
										diagnosis: patient.diagnosis || "",
										observations: "",
									}}
									onSubmit={onSubmit}
								>
									{(field) => (
										<div className="space-y-4">
											<FormTextArea
												field={field}
												name="diagnosis"
												label={
													<Text uuid="dialog.medical_certificate.diagnosis" />
												}
											/>
											<FormTextArea
												field={field}
												name="observations"
												label={
													<Text uuid="dialog.medical_certificate.observations" />
												}
											/>
											<div className="max-w-[200px]">
												<FormInput
													field={field}
													name="rest_days"
													type="number"
													label={textGet(
														"dialog.medical_certificate.rest_days",
													)}
												/>
											</div>
											<div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
												<FormCalendar
													field={field}
													name="rest_from"
													label={textGet(
														"dialog.medical_certificate.rest_from",
													)}
												/>
												<FormCalendar
													field={field}
													name="rest_to"
													label={textGet("dialog.medical_certificate.rest_to")}
												/>
											</div>

											{showEmailInput && (
												<div className="flex items-center gap-2">
													<Input
														type="email"
														value={email}
														onChange={(e) => setEmail(e.target.value)}
														placeholder={textGet(
															"dialog.medical_certificate.email_placeholder",
														)}
													/>
													<Button
														type="button"
														disabled={sending || !email}
														onClick={handleSendEmail}
													>
														{sending ? (
															<Loader2 className="h-4 w-4 animate-spin" />
														) : (
															<Mail className="h-4 w-4" />
														)}
													</Button>
												</div>
											)}

											<div className="flex flex-wrap gap-2 justify-end border-t pt-4">
												<Button
													type="button"
													variant="outline"
													disabled={!savedCertificateId || printing}
													onClick={handlePrint}
												>
													{printing ? (
														<Loader2 className="mr-2 h-4 w-4 animate-spin" />
													) : (
														<Printer className="mr-2 h-4 w-4" />
													)}
													<Text uuid="dialog.medical_certificate.print" />
												</Button>
												<Button
													type="button"
													variant="outline"
													disabled={!savedCertificateId}
													onClick={() => setShowEmailInput(true)}
												>
													<Mail className="mr-2 h-4 w-4" />
													<Text uuid="dialog.medical_certificate.send_email" />
												</Button>
												<Button type="submit" disabled={saving}>
													{saving ? (
														<Loader2 className="mr-2 h-4 w-4 animate-spin" />
													) : (
														<Save className="mr-2 h-4 w-4" />
													)}
													<Text uuid="dialog.medical_certificate.save" />
												</Button>
											</div>
										</div>
									)}
								</Form>
							</>
						)}
					</CardContent>
				</Card>
			</main>
		</DashboardLayout>
	);
}
