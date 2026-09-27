import { useText } from "@pengi/shared";
import { Badge, Form, FormTextArea, Text } from "@pengi/ui";
import { ChevronDown, Loader2 } from "lucide-react";
import React from "react";
import { type UseFormReturn, useFieldArray } from "react-hook-form";
import { useSearchParams } from "react-router";
import { z } from "zod";
import {
	createMedicalReport,
	downloadMedicalReportPdf,
	emailMedicalReport,
	getMedicalRecords,
	getPatientById,
	type MedicalRecord,
	type Patient,
} from "@/api/clinical-service";
import {
	type DocumentSignature,
	signMedicalReport,
} from "@/api/signature-service";
import { DocumentPageHeader } from "@/components/features/medical-documents/document-page-header";
import {
	DocumentWorkflow,
	type DocumentWorkflowProps,
	useSavedBaseline,
} from "@/components/features/medical-documents/document-workflow";
import { cn } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const BACKGROUND_FIELDS = ["app", "apf", "apqx", "allergies"] as const;

const SOAP_FIELDS = [
	{ name: "subjective", letter: "S" },
	{ name: "objective", letter: "O" },
	{ name: "assessment", letter: "A" },
	{ name: "plan", letter: "P" },
] as const;

/** Past this many consultations, only the most recent ones start expanded. */
const EXPANDED_BY_DEFAULT = 3;

const consultationSchema = z.object({
	medical_record_id: z.number(),
	date: z.string(),
	motive: z.string(),
	visit_type: z.string(),
	observation: z.string(),
	subjective: z.string(),
	objective: z.string(),
	assessment: z.string(),
	plan: z.string(),
	app: z.string(),
	apf: z.string(),
	apqx: z.string(),
	allergies: z.string(),
	diagnoses: z.array(z.object({ code: z.string(), title: z.string() })),
	vital_signs: z
		.object({
			weight: z.number().nullish(),
			height: z.number().nullish(),
			blood_pressure: z.string().optional(),
			temperature: z.number().nullish(),
			heart_rate: z.number().nullish(),
			o2_saturation: z.number().nullish(),
		})
		.nullable(),
	prescription: z
		.object({
			indications: z.string(),
			items: z.array(
				z.object({
					medication: z.string(),
					dose: z.string(),
					frequency: z.string(),
					duration: z.string(),
					notes: z.string().optional(),
				}),
			),
		})
		.nullable(),
});

const reportSchema = z.object({
	consultations: z.array(consultationSchema),
	plan: z.string(),
});

type ReportFormValues = z.input<typeof reportSchema>;
type ConsultationValues = z.input<typeof consultationSchema>;

function toConsultationValues(record: MedicalRecord): ConsultationValues {
	const vitals = record.vital_signs;
	const prescription = record.prescription;
	return {
		medical_record_id: record.ID,
		date: record.date,
		motive: record.motive,
		visit_type: record.visit_type ?? "",
		observation: record.observation ?? "",
		subjective: record.soap_record?.subjective ?? "",
		objective: record.soap_record?.objective ?? "",
		assessment: record.soap_record?.assessment ?? "",
		plan: record.soap_record?.plan ?? "",
		app: record.app ?? "",
		apf: record.apf ?? "",
		apqx: record.apqx ?? "",
		allergies: record.allergies ?? "",
		diagnoses: (record.diagnoses ?? []).map(({ code, title }) => ({
			code,
			title,
		})),
		vital_signs: vitals
			? {
					weight: vitals.weight,
					height: vitals.height,
					blood_pressure: vitals.blood_pressure,
					temperature: vitals.temperature,
					heart_rate: vitals.heart_rate,
					o2_saturation: vitals.o2_saturation,
				}
			: null,
		prescription:
			prescription &&
			(prescription.indications || (prescription.items?.length ?? 0) > 0)
				? {
						indications: prescription.indications ?? "",
						items: (prescription.items ?? []).map((item) => ({
							medication: item.medication,
							dose: item.dose,
							frequency: item.frequency,
							duration: item.duration,
							notes: item.notes,
						})),
					}
				: null,
	};
}

/** Borderless textarea that reads as document text until hovered or focused. */
const DOCUMENT_FIELD_CLASS =
	"min-h-0 -mx-2 w-[calc(100%+1rem)] resize-none border-transparent bg-transparent px-2 py-1.5 shadow-none hover:bg-muted/60 focus-visible:bg-background focus-visible:border-ring read-only:hover:bg-transparent dark:bg-transparent";

function VitalSigns({ vitals }: { vitals: ConsultationValues["vital_signs"] }) {
	const { textGet } = useText();
	if (!vitals) return null;
	const items = [
		{ key: "blood_pressure", value: vitals.blood_pressure, unit: "mmHg" },
		{ key: "heart_rate", value: vitals.heart_rate, unit: "bpm" },
		{ key: "temperature", value: vitals.temperature, unit: "°C" },
		{ key: "o2_saturation", value: vitals.o2_saturation, unit: "%" },
		{ key: "weight", value: vitals.weight, unit: "kg" },
		{ key: "height", value: vitals.height, unit: "cm" },
	].filter((item) => item.value != null && item.value !== "");
	if (items.length === 0) return null;

	return (
		<dl className="flex flex-wrap gap-x-5 gap-y-2 text-sm">
			{items.map((item) => (
				<div key={item.key} className="flex flex-col">
					<dt className="text-xs text-muted-foreground">
						{textGet(`view.medical_record.vital_signs.${item.key}`)}
					</dt>
					<dd className="font-medium tabular-nums">
						{item.value}
						<span className="ml-0.5 text-xs font-normal text-muted-foreground">
							{item.unit}
						</span>
					</dd>
				</div>
			))}
		</dl>
	);
}

function Prescription({
	prescription,
}: {
	prescription: ConsultationValues["prescription"];
}) {
	if (!prescription) return null;
	return (
		<section className="space-y-1.5">
			<h4 className="text-xs font-medium text-muted-foreground">
				<Text uuid="view.medical_record.prescription.title" />
			</h4>
			{prescription.items.length > 0 && (
				<ul className="space-y-1 text-sm">
					{prescription.items.map((item) => (
						<li
							key={`${item.medication}-${item.dose}-${item.frequency}`}
							className="flex flex-wrap gap-x-2"
						>
							<span className="font-medium">{item.medication}</span>
							<span className="text-muted-foreground">
								{[item.dose, item.frequency, item.duration, item.notes]
									.filter(Boolean)
									.join(", ")}
							</span>
						</li>
					))}
				</ul>
			)}
			{prescription.indications && (
				<p className="whitespace-pre-wrap text-sm">
					{prescription.indications}
				</p>
			)}
		</section>
	);
}

interface ConsultationEntryProps {
	field: UseFormReturn<ReportFormValues>;
	index: number;
	consultation: ConsultationValues;
	expanded: boolean;
	onToggle: () => void;
	readOnly: boolean;
	isLast: boolean;
}

function ConsultationEntry({
	field,
	index,
	consultation,
	expanded,
	onToggle,
	readOnly,
	isLast,
}: ConsultationEntryProps) {
	const { textGet } = useText();
	const date = new Date(consultation.date);
	const bodyId = `consultation-${consultation.medical_record_id}`;
	const placeholder = textGet("view.medical_record.not_registered");
	const background = BACKGROUND_FIELDS.filter((key) => consultation[key]);

	return (
		<li className="grid grid-cols-[3.5rem_minmax(0,1fr)] gap-x-4 sm:grid-cols-[4.5rem_minmax(0,1fr)]">
			<div className="pt-0.5 text-right">
				<div className="text-2xl font-semibold leading-none tabular-nums">
					{date.getDate()}
				</div>
				<div className="mt-1 text-xs text-muted-foreground capitalize">
					{date.toLocaleDateString("es-EC", {
						month: "short",
						year: "numeric",
					})}
				</div>
			</div>

			<div
				className={cn(
					"relative border-l pl-5 sm:pl-6",
					isLast ? "pb-2" : "pb-8",
				)}
			>
				<span
					aria-hidden
					className="absolute top-1.5 -left-[5px] size-2.5 rounded-full border-2 border-background bg-primary"
				/>

				<button
					type="button"
					onClick={onToggle}
					aria-expanded={expanded}
					aria-controls={bodyId}
					className="-mx-2 -my-1 flex w-[calc(100%+1rem)] items-start gap-2 rounded-md px-2 py-1 text-left hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
				>
					<span className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
						<span className="font-medium">
							{consultation.motive || placeholder}
						</span>
						{consultation.visit_type && (
							<Badge variant="secondary" className="font-normal">
								{textGet(
									`form.create_medical_record.visit_type.${consultation.visit_type}`,
								)}
							</Badge>
						)}
					</span>
					<ChevronDown
						className={cn(
							"mt-0.5 size-4 shrink-0 text-muted-foreground transition-transform motion-reduce:transition-none",
							expanded && "rotate-180",
						)}
					/>
				</button>

				{!expanded && consultation.diagnoses.length > 0 && (
					<p className="mt-1 truncate text-sm text-muted-foreground">
						{consultation.diagnoses.map((d) => d.code).join(", ")}
					</p>
				)}

				{expanded && (
					<div id={bodyId} className="mt-4 space-y-5">
						<VitalSigns vitals={consultation.vital_signs} />

						{consultation.diagnoses.length > 0 && (
							<ul className="flex flex-wrap gap-1.5">
								{consultation.diagnoses.map((d) => (
									<li key={d.code}>
										<Badge
											variant="outline"
											className="gap-1.5 py-1 font-normal"
										>
											<span className="font-medium text-primary">{d.code}</span>
											<span>{d.title}</span>
										</Badge>
									</li>
								))}
							</ul>
						)}

						<div className="grid grid-cols-[1.75rem_minmax(0,1fr)] gap-x-3">
							<div className="col-start-2 space-y-0.5">
								<label
									htmlFor={`consultations.${index}.observation`}
									className="text-xs font-medium text-muted-foreground"
								>
									<Text uuid="view.medical_record.observation" />
								</label>
								<FormTextArea
									field={field}
									name={`consultations.${index}.observation`}
									placeholder={placeholder}
									readOnly={readOnly}
									className={DOCUMENT_FIELD_CLASS}
								/>
							</div>
						</div>

						<div className="space-y-3">
							{SOAP_FIELDS.map(({ name, letter }) => {
								const id = `consultations.${index}.${name}` as const;
								return (
									<div
										key={name}
										className="grid grid-cols-[1.75rem_minmax(0,1fr)] gap-x-3"
									>
										<span
											aria-hidden
											className="mt-1 grid size-7 place-items-center rounded-md bg-primary/10 text-sm font-semibold text-primary"
										>
											{letter}
										</span>
										<div className="space-y-0.5">
											<label
												htmlFor={id}
												className="text-xs font-medium text-muted-foreground"
											>
												<Text uuid={`dialog.medical_report.soap.${name}`} />
											</label>
											<FormTextArea
												field={field}
												name={id}
												placeholder={placeholder}
												readOnly={readOnly}
												className={DOCUMENT_FIELD_CLASS}
											/>
										</div>
									</div>
								);
							})}
						</div>

						<Prescription prescription={consultation.prescription} />

						{background.length > 0 && (
							<section className="space-y-2">
								<h4 className="text-xs font-medium text-muted-foreground">
									<Text uuid="dialog.medical_report.background" />
								</h4>
								<div className="grid gap-x-6 gap-y-2 md:grid-cols-2">
									{background.map((key) => (
										<div key={key} className="space-y-0.5">
											<label
												htmlFor={`consultations.${index}.${key}`}
												className="text-xs text-muted-foreground"
											>
												{textGet(`form.create_medical_record.${key}`)}
											</label>
											<FormTextArea
												field={field}
												name={`consultations.${index}.${key}`}
												readOnly={readOnly}
												className={DOCUMENT_FIELD_CLASS}
											/>
										</div>
									))}
								</div>
							</section>
						)}
					</div>
				)}
			</div>
		</li>
	);
}

interface ReportEditorProps
	extends Omit<DocumentWorkflowProps, "isDirty" | "saveLabel"> {
	field: UseFormReturn<ReportFormValues>;
	savedVersion: number;
}

function ReportEditor({ field, savedVersion, ...actions }: ReportEditorProps) {
	const { fields } = useFieldArray({
		control: field.control,
		name: "consultations",
	});
	const readOnly = Boolean(actions.signature);

	useSavedBaseline(field, savedVersion);

	// Keyed by medical record, not field id: field.reset() regenerates field ids.
	const [expanded, setExpanded] = React.useState<Set<number>>(
		() =>
			new Set(
				fields
					.slice(Math.max(0, fields.length - EXPANDED_BY_DEFAULT))
					.map((f) => f.medical_record_id),
			),
	);

	function toggle(id: number) {
		setExpanded((current) => {
			const next = new Set(current);
			if (next.has(id)) next.delete(id);
			else next.add(id);
			return next;
		});
	}

	return (
		<div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_18rem]">
			<div className="space-y-8">
				<section className="space-y-5">
					<h2 className="text-base font-semibold">
						<Text uuid="dialog.medical_report.consultations" />
					</h2>
					{fields.length === 0 ? (
						<p className="text-sm text-muted-foreground">
							<Text uuid="dialog.medical_report.no_consultations" />
						</p>
					) : (
						<ol>
							{fields.map((consultation, index) => (
								<ConsultationEntry
									key={consultation.id}
									field={field}
									index={index}
									consultation={consultation}
									expanded={expanded.has(consultation.medical_record_id)}
									onToggle={() => toggle(consultation.medical_record_id)}
									readOnly={readOnly}
									isLast={index === fields.length - 1}
								/>
							))}
						</ol>
					)}
				</section>

				<section className="space-y-2 border-t pt-6">
					<label htmlFor="plan" className="text-base font-semibold">
						<Text uuid="dialog.medical_report.general_plan" />
					</label>
					<FormTextArea
						field={field}
						name="plan"
						readOnly={readOnly}
						className="min-h-24"
					/>
				</section>

				{readOnly && (
					<p className="text-sm text-muted-foreground">
						<Text uuid="medical_document.signed_readonly" />
					</p>
				)}
			</div>

			<DocumentWorkflow
				{...actions}
				saveLabel={<Text uuid="dialog.medical_report.save_report" />}
				isDirty={field.formState.isDirty}
			/>
		</div>
	);
}

export default function GenerateMedicalReportPage() {
	const [searchParams] = useSearchParams();
	const patientId = Number(searchParams.get("patient_id"));

	const [patient, setPatient] = React.useState<Patient | null>(null);
	const [loadingRecords, setLoadingRecords] = React.useState(true);
	const [saving, setSaving] = React.useState(false);
	const [printing, setPrinting] = React.useState(false);
	const [savedReportId, setSavedReportId] = React.useState<number | null>(null);
	const [savedVersion, setSavedVersion] = React.useState(0);
	const [signature, setSignature] = React.useState<DocumentSignature | null>(
		null,
	);
	const [defaultValues, setDefaultValues] =
		React.useState<ReportFormValues | null>(null);

	React.useEffect(() => {
		if (!patientId) return;
		Promise.all([
			getPatientById(patientId),
			getMedicalRecords(patientId, { page: 1, limit: 100 }),
		]).then(([patientRes, recordsRes]) => {
			if (patientRes.success && patientRes.data) {
				setPatient(patientRes.data);
			}
			const records = recordsRes.success ? (recordsRes.data?.items ?? []) : [];
			const sorted = [...records].sort(
				(a, b) => new Date(a.date).getTime() - new Date(b.date).getTime(),
			);
			setDefaultValues({
				consultations: sorted.map(toConsultationValues),
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
			setSavedVersion((v) => v + 1);
			setSignature(null);
		}
		setSaving(false);
	}

	async function handleSign() {
		if (!savedReportId) return false;
		const res = await signMedicalReport(savedReportId);
		if (res.success && res.data) setSignature(res.data);
		return res.success;
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

	async function handleSendEmail(email: string) {
		if (!savedReportId || !email) return;
		await emailMedicalReport(savedReportId, email);
	}

	return (
		<DashboardLayout>
			<div className="mx-auto w-full max-w-6xl space-y-6">
				<DocumentPageHeader
					patientId={patientId}
					patient={patient}
					title="dialog.medical_report.title"
					description="dialog.medical_report.description"
				/>

				{loadingRecords || !defaultValues ? (
					<div className="flex justify-center py-16">
						<Loader2 className="size-6 animate-spin text-muted-foreground" />
					</div>
				) : (
					<Form
						schema={reportSchema}
						defaultValues={defaultValues}
						onSubmit={onSubmit}
					>
						{(field) => (
							<ReportEditor
								field={field}
								savedVersion={savedVersion}
								saving={saving}
								printing={printing}
								isSaved={savedReportId !== null}
								signature={signature}
								patientEmail={patient?.email ?? ""}
								onSign={handleSign}
								onPrint={handlePrint}
								onSendEmail={handleSendEmail}
							/>
						)}
					</Form>
				)}
			</div>
		</DashboardLayout>
	);
}
