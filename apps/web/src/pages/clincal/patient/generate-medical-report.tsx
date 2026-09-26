import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardHeader,
	CardTitle,
	Form,
	FormTextArea,
	Input,
	Text,
} from "@pengi/ui";
import { ArrowLeft, Loader2, Mail, Printer, Save } from "lucide-react";
import React from "react";
import { type UseFormReturn, useFieldArray } from "react-hook-form";
import { useNavigate, useSearchParams } from "react-router";
import { z } from "zod";
import {
	createMedicalReport,
	downloadMedicalReportPdf,
	emailMedicalReport,
	getMedicalRecords,
	getPatientById,
	type Patient,
} from "@/api/clinical-service";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const consultationSchema = z.object({
	medical_record_id: z.number(),
	date: z.string(),
	motive: z.string(),
	summary: z.string(),
});

const reportSchema = z.object({
	consultations: z.array(consultationSchema),
	plan: z.string(),
});

type ReportFormValues = z.input<typeof reportSchema>;

interface ReportFormFieldsProps {
	field: UseFormReturn<ReportFormValues>;
	saving: boolean;
	printing: boolean;
	sending: boolean;
	savedReportId: number | null;
	showEmailInput: boolean;
	email: string;
	onEmailChange: (value: string) => void;
	onPrint: () => void;
	onShowEmailInput: () => void;
	onSendEmail: () => void;
}

function ReportFormFields({
	field,
	saving,
	printing,
	sending,
	savedReportId,
	showEmailInput,
	email,
	onEmailChange,
	onPrint,
	onShowEmailInput,
	onSendEmail,
}: ReportFormFieldsProps) {
	const { textGet } = useText();
	const { fields } = useFieldArray({
		control: field.control,
		name: "consultations",
	});

	return (
		<div className="space-y-4">
			<div className="space-y-3">
				<h4 className="text-sm font-semibold">
					<Text uuid="dialog.medical_report.consultations" />
				</h4>
				{fields.length === 0 && (
					<p className="text-sm text-muted-foreground">
						<Text uuid="dialog.medical_report.no_consultations" />
					</p>
				)}
				{fields.map((consultation, index) => (
					<div
						key={consultation.id}
						className="rounded-md border p-3 space-y-2"
					>
						<div className="text-xs text-muted-foreground">
							{new Date(consultation.date).toLocaleDateString("es-EC")} —{" "}
							{consultation.motive}
						</div>
						<FormTextArea
							field={field}
							name={`consultations.${index}.summary`}
							label={<Text uuid="dialog.medical_report.summary" />}
						/>
					</div>
				))}
			</div>

			<FormTextArea
				field={field}
				name="plan"
				label={<Text uuid="dialog.medical_report.plan" />}
			/>

			{showEmailInput && (
				<div className="flex items-center gap-2">
					<Input
						type="email"
						value={email}
						onChange={(e) => onEmailChange(e.target.value)}
						placeholder={textGet("dialog.medical_report.email_placeholder")}
					/>
					<Button
						type="button"
						disabled={sending || !email}
						onClick={onSendEmail}
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
					disabled={!savedReportId || printing}
					onClick={onPrint}
				>
					{printing ? (
						<Loader2 className="mr-2 h-4 w-4 animate-spin" />
					) : (
						<Printer className="mr-2 h-4 w-4" />
					)}
					<Text uuid="dialog.medical_report.print" />
				</Button>
				<Button
					type="button"
					variant="outline"
					disabled={!savedReportId}
					onClick={onShowEmailInput}
				>
					<Mail className="mr-2 h-4 w-4" />
					<Text uuid="dialog.medical_report.send_email" />
				</Button>
				<Button type="submit" disabled={saving}>
					{saving ? (
						<Loader2 className="mr-2 h-4 w-4 animate-spin" />
					) : (
						<Save className="mr-2 h-4 w-4" />
					)}
					<Text uuid="dialog.medical_report.save" />
				</Button>
			</div>
		</div>
	);
}

export default function GenerateMedicalReportPage() {
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const patientId = Number(searchParams.get("patient_id"));

	const [patient, setPatient] = React.useState<Patient | null>(null);
	const [loadingRecords, setLoadingRecords] = React.useState(true);
	const [saving, setSaving] = React.useState(false);
	const [printing, setPrinting] = React.useState(false);
	const [sending, setSending] = React.useState(false);
	const [savedReportId, setSavedReportId] = React.useState<number | null>(null);
	const [showEmailInput, setShowEmailInput] = React.useState(false);
	const [email, setEmail] = React.useState("");
	const [defaultValues, setDefaultValues] = React.useState<z.input<
		typeof reportSchema
	> | null>(null);

	React.useEffect(() => {
		if (!patientId) return;
		Promise.all([
			getPatientById(patientId),
			getMedicalRecords(patientId, { page: 1, limit: 100 }),
		]).then(([patientRes, recordsRes]) => {
			if (patientRes.success && patientRes.data) {
				setPatient(patientRes.data);
				setEmail(patientRes.data.email ?? "");
			}
			const records = recordsRes.success ? (recordsRes.data?.items ?? []) : [];
			const sorted = [...records].sort(
				(a, b) => new Date(a.date).getTime() - new Date(b.date).getTime(),
			);
			setDefaultValues({
				consultations: sorted.map((record) => ({
					medical_record_id: record.ID,
					date: record.date,
					motive: record.motive,
					summary: record.soap_record?.plan || record.observation || "",
				})),
				plan: "",
			});
			setLoadingRecords(false);
		});
	}, [patientId]);

	async function onSubmit(values: z.infer<typeof reportSchema>) {
		setSaving(true);
		const res = await createMedicalReport(patientId, values);
		if (res.success && res.data) {
			setSavedReportId(res.data.ID);
		}
		setSaving(false);
	}

	async function handlePrint() {
		if (!savedReportId) return;
		setPrinting(true);
		const res = await downloadMedicalReportPdf(savedReportId);
		if (res.success && res.data) {
			const url = window.URL.createObjectURL(res.data);
			window.open(url, "_blank");
		}
		setPrinting(false);
	}

	async function handleSendEmail() {
		if (!savedReportId || !email) return;
		setSending(true);
		await emailMedicalReport(savedReportId, email);
		setSending(false);
	}

	const fullName = patient
		? patient.full_name || `${patient.last_name} ${patient.first_name}`
		: "";

	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<Card className="max-w-4xl mx-auto w-full">
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
							<Text uuid="dialog.medical_report.title" />
						</CardTitle>
						<p className="text-sm text-muted-foreground">
							<Text uuid="dialog.medical_report.description" />
						</p>
					</CardHeader>
					<CardContent className="space-y-4">
						{patient && (
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
						)}

						{loadingRecords || !defaultValues ? (
							<div className="flex justify-center py-8">
								<Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
							</div>
						) : (
							<Form
								schema={reportSchema}
								defaultValues={defaultValues}
								onSubmit={onSubmit}
							>
								{(field) => (
									<ReportFormFields
										field={field}
										saving={saving}
										printing={printing}
										sending={sending}
										savedReportId={savedReportId}
										showEmailInput={showEmailInput}
										email={email}
										onEmailChange={setEmail}
										onPrint={handlePrint}
										onShowEmailInput={() => setShowEmailInput(true)}
										onSendEmail={handleSendEmail}
									/>
								)}
							</Form>
						)}
					</CardContent>
				</Card>
			</main>
		</DashboardLayout>
	);
}
