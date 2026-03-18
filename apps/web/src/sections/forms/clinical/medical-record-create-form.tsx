import {
	Activity,
	AlertTriangle,
	BookOpen,
	Calendar,
	ChevronLeft,
	ChevronRight,
	Eye,
	Pill,
	Plus,
	Stethoscope,
	Trash,
} from "lucide-react";
import React from "react";
import { type UseFormReturn, useFieldArray, useWatch } from "react-hook-form";
import { useNavigate, useSearchParams } from "react-router";
import { z } from "zod";
import {
	type CreateMedicalRecordPayload,
	createMedicalRecord,
	getMedicalRecords,
	type MedicalRecord,
} from "@/api/clinical-service";
import { Form } from "@/components/forms/form";
import { FormCalendar } from "@/components/forms/form-calendar";
import { FormIcd11Select } from "@/components/forms/form-icd11-select";
import { FormInput } from "@/components/forms/form-input";
import { FormTextArea } from "@/components/forms/form-textarea";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Text } from "@/components/ui/text";
import useTenantSettings from "@/hooks/use-tenant-settings";
import { useText } from "@/hooks/use-text";
import { usePatientStore } from "@/store/patient-store";

type PrescriptionMode = "text" | "structured";

const prescriptionItemSchema = z.object({
	medication: z.string().min(1, "Campo requerido"),
	dose: z.string().min(1, "Campo requerido"),
	frequency: z.string().min(1, "Campo requerido"),
	duration: z.string().min(1, "Campo requerido"),
	notes: z.string().optional(),
});

const formSchema = z.object({
	date: z.date({ error: "Campo requerido" }),
	motive: z
		.string({ error: "Campo requerido" })
		.min(1, "Campo requerido")
		.max(100, "Máximo 100 caracteres"),
	observation: z.string().optional(),
	soap_record: z.object({
		subjective: z
			.string({ error: "Campo requerido" })
			.min(1, "Campo requerido"),
		objective: z.string({ error: "Campo requerido" }).min(1, "Campo requerido"),
		assessment: z
			.string({ error: "Campo requerido" })
			.min(1, "Campo requerido"),
		plan: z.string({ error: "Campo requerido" }).min(1, "Campo requerido"),
	}),
	prescription: z
		.object({
			content: z.string().optional(),
			indications: z.string().optional(),
			items: z.array(prescriptionItemSchema).optional(),
		})
		.optional(),
	vital_signs: z
		.object({
			weight: z.coerce.number().positive().optional().nullable(),
			height: z.coerce.number().positive().optional().nullable(),
			blood_pressure: z.string().optional(),
			temperature: z.coerce.number().positive().optional().nullable(),
			heart_rate: z.coerce.number().int().positive().optional().nullable(),
			o2_saturation: z.coerce
				.number()
				.int()
				.min(0)
				.max(100)
				.optional()
				.nullable(),
		})
		.optional(),
	diagnoses: z
		.array(z.object({ code: z.string(), title: z.string() }))
		.optional(),
});

const TABS = ["consulta", "soap", "complementario"] as const;
type TabId = (typeof TABS)[number];

const CreateMedicalRecordForm = () => {
	const [loading, setLoading] = React.useState(false);
	const [prescriptionMode, setPrescriptionMode] =
		React.useState<PrescriptionMode>("text");
	const [lastRecord, setLastRecord] = React.useState<MedicalRecord | null>(
		null,
	);
	const [previewOpen, setPreviewOpen] = React.useState(false);
	const { textGet } = useText();
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const patientId = searchParams.get("patient_id");
	const { patient } = usePatientStore();
	const allergies = parseAllergies(patient?.allergies);

	React.useEffect(() => {
		if (!patientId) return;
		getMedicalRecords(Number(patientId), { page: 1, limit: 1 }).then((res) => {
			if (res.success && res.data && res.data.items.length > 0) {
				setLastRecord(res.data.items[0]);
			}
		});
	}, [patientId]);

	return (
		<>
			<Form
				schema={formSchema}
				onSubmit={onSubmit}
				defaultValues={{
					date: new Date(),
					prescription: { items: [] },
				}}
			>
				{(field) => (
					<FormInner
						field={field}
						loading={loading}
						textGet={textGet}
						prescriptionMode={prescriptionMode}
						onPrescriptionModeChange={setPrescriptionMode}
						allergies={allergies}
						lastRecord={lastRecord}
						onPreviewLastRecord={() => setPreviewOpen(true)}
					/>
				)}
			</Form>
			{lastRecord && (
				<LastRecordDialog
					record={lastRecord}
					open={previewOpen}
					onClose={() => setPreviewOpen(false)}
					textGet={textGet}
				/>
			)}
		</>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!patientId) return;
		setLoading(true);

		let prescription: CreateMedicalRecordPayload["prescription"];

		if (prescriptionMode === "text") {
			const hasText =
				values.prescription?.content || values.prescription?.indications;
			if (hasText) {
				prescription = {
					content: values.prescription?.content || "",
					indications: values.prescription?.indications || "",
					items: [],
				};
			}
		} else {
			const items = values.prescription?.items ?? [];
			if (items.length > 0) {
				const content = items.map((item) => item.medication).join("\n");
				const indications = items
					.map((item) =>
						[item.dose, item.frequency, item.duration, item.notes]
							.filter(Boolean)
							.join(" — "),
					)
					.join("\n");
				prescription = { content, indications, items };
			}
		}

		const payload = {
			patient_id: Number(patientId),
			date: values.date.toISOString(),
			motive: values.motive,
			observation: values.observation || "",
			soap_record: values.soap_record,
			prescription,
			diagnoses: values.diagnoses ?? [],
			vital_signs: values.vital_signs
				? {
						weight: values.vital_signs.weight ?? null,
						height: values.vital_signs.height ?? null,
						blood_pressure: values.vital_signs.blood_pressure || undefined,
						temperature: values.vital_signs.temperature ?? null,
						heart_rate: values.vital_signs.heart_rate ?? null,
						o2_saturation: values.vital_signs.o2_saturation ?? null,
					}
				: undefined,
		};

		try {
			const res = await createMedicalRecord(payload);
			if (res.success) {
				navigate(`/clinical/medical-records/${patientId}`);
			}
		} finally {
			setLoading(false);
		}
	}
};

function parseAllergies(allergies?: string): string[] {
	if (!allergies?.trim()) return [];
	return allergies
		.split(",")
		.map((a) => a.trim())
		.filter(Boolean);
}

function FormInner({
	field,
	loading,
	textGet,
	prescriptionMode,
	onPrescriptionModeChange,
	allergies,
	lastRecord,
	onPreviewLastRecord,
}: {
	field: UseFormReturn<
		z.input<typeof formSchema>,
		unknown,
		z.infer<typeof formSchema>
	>;
	loading: boolean;
	textGet: (key: string) => string;
	prescriptionMode: PrescriptionMode;
	onPrescriptionModeChange: (mode: PrescriptionMode) => void;
	allergies: string[];
	lastRecord: MedicalRecord | null;
	onPreviewLastRecord: () => void;
}) {
	const [activeTab, setActiveTab] = React.useState<TabId>("consulta");
	const [footerStuck, setFooterStuck] = React.useState(false);
	const sentinelRef = React.useRef<HTMLDivElement>(null);

	React.useEffect(() => {
		const sentinel = sentinelRef.current;
		if (!sentinel) return;
		const observer = new IntersectionObserver(
			([entry]) => setFooterStuck(!entry.isIntersecting),
			{ threshold: 0 },
		);
		observer.observe(sentinel);
		return () => observer.disconnect();
	}, []);

	const { fields, append, remove } = useFieldArray({
		name: "prescription.items",
		control: field.control,
	});
	const { settings } = useTenantSettings();
	const { show_vital_signs, show_diagnoses, diagnosis_system } =
		settings.clinical;

	const watchedMotive = useWatch({ control: field.control, name: "motive" });
	const watchedSoap = useWatch({
		control: field.control,
		name: "soap_record",
	});

	const watchedPrescription = useWatch({
		control: field.control,
		name: "prescription",
	});
	const watchedDiagnoses = useWatch({
		control: field.control,
		name: "diagnoses",
	});

	const tabDone: Record<TabId, boolean> = {
		consulta: !!watchedMotive,
		soap: !!(
			watchedSoap?.subjective &&
			watchedSoap?.objective &&
			watchedSoap?.assessment &&
			watchedSoap?.plan
		),
		complementario: !!(
			(watchedDiagnoses && watchedDiagnoses.length > 0) ||
			watchedPrescription?.content ||
			watchedPrescription?.indications ||
			(watchedPrescription?.items && watchedPrescription.items.length > 0)
		),
	};

	function goTo(tab: TabId) {
		setActiveTab(tab);
	}

	function goNext() {
		const idx = TABS.indexOf(activeTab);
		if (idx < TABS.length - 1) setActiveTab(TABS[idx + 1]);
	}

	function goPrev() {
		const idx = TABS.indexOf(activeTab);
		if (idx > 0) setActiveTab(TABS[idx - 1]);
	}

	const isFirst = activeTab === TABS[0];
	const isLast = activeTab === TABS[TABS.length - 1];

	return (
		<div className="max-w-5xl mx-auto flex flex-col gap-4">
			{/* Sticky compact allergy banner */}
			{allergies.length > 0 && (
				<div className="sticky top-0 z-20 -mx-1">
					<div className="flex items-center gap-3 bg-amber-50 dark:bg-amber-950/30 border border-amber-300 dark:border-amber-700 rounded-lg px-4 py-2 shadow-sm">
						<AlertTriangle className="h-4 w-4 text-amber-600 shrink-0" />
						<span className="text-sm font-semibold text-amber-800 dark:text-amber-200 shrink-0">
							<Text uuid="form.create_medical_record.allergy_alert.title" />:
						</span>
						<div className="flex flex-wrap gap-1.5 min-w-0">
							{allergies.map((a) => (
								<Badge
									key={a}
									variant="outline"
									className="bg-amber-100 dark:bg-amber-900/40 text-amber-800 dark:text-amber-300 border-amber-400 text-xs"
								>
									{a}
								</Badge>
							))}
						</div>
					</div>
				</div>
			)}

			{/* Tab navigation */}
			<Tabs
				value={activeTab}
				onValueChange={(v) => setActiveTab(v as TabId)}
				className="w-full mb-16"
			>
				<TabsList className="w-full">
					<TabsTrigger value="consulta" className="flex-1 gap-1.5">
						<Calendar className="h-3.5 w-3.5 shrink-0" />
						<Text uuid="form.create_medical_record.tab.consulta" />
					</TabsTrigger>
					<TabsTrigger value="soap" className="flex-1 gap-1.5">
						<Stethoscope className="h-3.5 w-3.5 shrink-0" />
						SOAP
					</TabsTrigger>
					<TabsTrigger value="complementario" className="flex-1 gap-1.5">
						<BookOpen className="h-3.5 w-3.5 shrink-0" />
						<Text uuid="form.create_medical_record.tab.complementario" />
					</TabsTrigger>
				</TabsList>

				{/* ── Tab 1: Consulta (General + Vitals) ── */}
				<TabsContent value="consulta" className="mt-4 space-y-4 pb-4">
					<Card>
						<CardHeader>
							<CardTitle>
								<Text uuid="form.create_medical_record.title" />
							</CardTitle>
							<CardDescription>
								<Text uuid="form.create_medical_record.description" />
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-4">
							<div className="grid md:grid-cols-2 grid-cols-1 gap-4">
								<FormCalendar
									field={field}
									name="date"
									label={textGet("form.create_medical_record.date")}
								/>
							</div>
							<FormTextArea
								field={field}
								name="motive"
								label={<Text uuid="form.create_medical_record.motive" />}
								placeholder={textGet(
									"form.create_medical_record.motive.placeholder",
								)}
								description={textGet(
									"form.create_medical_record.motive.description",
								)}
								rows={3}
							/>
							<FormTextArea
								field={field}
								name="observation"
								label={<Text uuid="form.create_medical_record.observation" />}
								placeholder={textGet(
									"form.create_medical_record.observation.placeholder",
								)}
								isOptional
								rows={2}
							/>
						</CardContent>
					</Card>

					{show_vital_signs && (
						<Card className="border-l-4 border-l-rose-500">
							<CardHeader>
								<div className="flex items-center gap-3">
									<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-rose-500 text-white">
										<Activity className="h-4 w-4" />
									</div>
									<div>
										<CardTitle className="text-base">
											<Text uuid="form.create_medical_record.vital_signs.title" />
										</CardTitle>
										<p className="text-sm text-muted-foreground">
											<Text uuid="form.create_medical_record.vital_signs.description" />
										</p>
									</div>
								</div>
							</CardHeader>
							<CardContent>
								<div className="grid md:grid-cols-3 grid-cols-2 gap-4">
									<FormInput
										field={field}
										name="vital_signs.weight"
										type="number"
										step="0.1"
										label={textGet(
											"form.create_medical_record.vital_signs.weight",
										)}
										placeholder="0.0"
										endAddon={
											<span className="text-muted-foreground text-sm px-2">
												kg
											</span>
										}
										isOptional
									/>
									<FormInput
										field={field}
										name="vital_signs.height"
										type="number"
										step="0.1"
										label={textGet(
											"form.create_medical_record.vital_signs.height",
										)}
										placeholder="0.0"
										endAddon={
											<span className="text-muted-foreground text-sm px-2">
												cm
											</span>
										}
										isOptional
									/>
									<FormInput
										field={field}
										name="vital_signs.blood_pressure"
										label={textGet(
											"form.create_medical_record.vital_signs.blood_pressure",
										)}
										placeholder="120/80"
										endAddon={
											<span className="text-muted-foreground text-sm px-2">
												mmHg
											</span>
										}
										isOptional
									/>
									<FormInput
										field={field}
										name="vital_signs.temperature"
										type="number"
										step="0.1"
										label={textGet(
											"form.create_medical_record.vital_signs.temperature",
										)}
										placeholder="36.5"
										endAddon={
											<span className="text-muted-foreground text-sm px-2">
												°C
											</span>
										}
										isOptional
									/>
									<FormInput
										field={field}
										name="vital_signs.heart_rate"
										type="number"
										label={textGet(
											"form.create_medical_record.vital_signs.heart_rate",
										)}
										placeholder="70"
										endAddon={
											<span className="text-muted-foreground text-sm px-2">
												bpm
											</span>
										}
										isOptional
									/>
									<FormInput
										field={field}
										name="vital_signs.o2_saturation"
										type="number"
										label={textGet(
											"form.create_medical_record.vital_signs.o2_saturation",
										)}
										placeholder="98"
										endAddon={
											<span className="text-muted-foreground text-sm px-2">
												%
											</span>
										}
										isOptional
									/>
								</div>
							</CardContent>
						</Card>
					)}
				</TabsContent>

				{/* ── Tab 2: SOAP ── */}
				<TabsContent value="soap" className="mt-4 pb-4">
					{/* SOAP step indicator */}
					<div className="flex items-center gap-2 mb-5">
						{(
							[
								{
									letter: "S",
									color: "bg-primary",
									key: "subjective",
									label: textGet(
										"form.create_medical_record.soap.subjective.title",
									),
								},
								{
									letter: "O",
									color: "bg-emerald-500",
									key: "objective",
									label: textGet(
										"form.create_medical_record.soap.objective.title",
									),
								},
								{
									letter: "A",
									color: "bg-orange-500",
									key: "assessment",
									label: textGet(
										"form.create_medical_record.soap.assessment.title",
									),
								},
								{
									letter: "P",
									color: "bg-teal-500",
									key: "plan",
									label: textGet(
										"form.create_medical_record.soap.plan.title",
									),
								},
							] as const
						).map((step, i) => (
							<React.Fragment key={step.key}>
								<div className="flex items-center gap-2">
									<div
										className={`h-7 w-7 rounded-full ${step.color} text-white flex items-center justify-center text-xs font-bold`}
									>
										{step.letter}
									</div>
									<span className="text-xs text-muted-foreground hidden sm:inline">
										{step.label}
									</span>
								</div>
								{i < 3 && (
									<div className="flex-1 h-px bg-border max-w-8" />
								)}
							</React.Fragment>
						))}
					</div>

					<div className="space-y-4">
						{/* S — Subjective */}
						<Card className="border-l-4 border-l-primary">
							<CardHeader className="pb-3">
								<div className="flex items-center gap-3">
									<div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground text-sm font-bold">
										S
									</div>
									<div>
										<CardTitle className="text-base">
											<Text uuid="form.create_medical_record.soap.subjective.title" />
										</CardTitle>
										<p className="text-xs text-muted-foreground">
											<Text uuid="form.create_medical_record.soap.subjective.description" />
										</p>
									</div>
								</div>
							</CardHeader>
							<CardContent>
								<FormTextArea
									field={field}
									name="soap_record.subjective"
									label={
										<Text uuid="form.create_medical_record.soap.subjective" />
									}
									placeholder={textGet(
										"form.create_medical_record.soap.subjective.placeholder",
									)}
									rows={4}
								/>
							</CardContent>
						</Card>

						{/* O — Objective */}
						<Card className="border-l-4 border-l-emerald-500">
							<CardHeader className="pb-3">
								<div className="flex items-center gap-3">
									<div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500 text-white text-sm font-bold">
										O
									</div>
									<div>
										<CardTitle className="text-base">
											<Text uuid="form.create_medical_record.soap.objective.title" />
										</CardTitle>
										<p className="text-xs text-muted-foreground">
											<Text uuid="form.create_medical_record.soap.objective.description" />
										</p>
									</div>
								</div>
							</CardHeader>
							<CardContent>
								<FormTextArea
									field={field}
									name="soap_record.objective"
									label={
										<Text uuid="form.create_medical_record.soap.objective" />
									}
									placeholder={textGet(
										"form.create_medical_record.soap.objective.placeholder",
									)}
									rows={4}
								/>
							</CardContent>
						</Card>

						{/* A — Assessment */}
						<Card className="border-l-4 border-l-orange-500">
							<CardHeader className="pb-3">
								<div className="flex items-center gap-3">
									<div className="flex h-8 w-8 items-center justify-center rounded-lg bg-orange-500 text-white text-sm font-bold">
										A
									</div>
									<div>
										<CardTitle className="text-base">
											<Text uuid="form.create_medical_record.soap.assessment.title" />
										</CardTitle>
										<p className="text-xs text-muted-foreground">
											<Text uuid="form.create_medical_record.soap.assessment.description" />
										</p>
									</div>
								</div>
							</CardHeader>
							<CardContent>
								<FormTextArea
									field={field}
									name="soap_record.assessment"
									label={
										<Text uuid="form.create_medical_record.soap.assessment" />
									}
									placeholder={textGet(
										"form.create_medical_record.soap.assessment.placeholder",
									)}
									rows={4}
								/>
							</CardContent>
						</Card>

						{/* P — Plan */}
						<Card className="border-l-4 border-l-teal-500">
							<CardHeader className="pb-3">
								<div className="flex items-center gap-3">
									<div className="flex h-8 w-8 items-center justify-center rounded-lg bg-teal-500 text-white text-sm font-bold">
										P
									</div>
									<div>
										<CardTitle className="text-base">
											<Text uuid="form.create_medical_record.soap.plan.title" />
										</CardTitle>
										<p className="text-xs text-muted-foreground">
											<Text uuid="form.create_medical_record.soap.plan.description" />
										</p>
									</div>
								</div>
							</CardHeader>
							<CardContent>
								<FormTextArea
									field={field}
									name="soap_record.plan"
									label={<Text uuid="form.create_medical_record.soap.plan" />}
									placeholder={textGet(
										"form.create_medical_record.soap.plan.placeholder",
									)}
									rows={4}
								/>
							</CardContent>
						</Card>
					</div>
				</TabsContent>

				{/* ── Tab 3: Complementario (Diagnoses + Prescription) ── */}
				<TabsContent value="complementario" className="mt-4 space-y-4 pb-4">
					{show_diagnoses && (
						<Card className="border-l-4 border-l-blue-500">
							<CardHeader>
								<div className="flex items-center gap-3">
									<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-blue-500 text-white">
										<BookOpen className="h-4 w-4" />
									</div>
									<div>
										<CardTitle className="text-base flex items-center gap-2">
											<Text uuid="form.create_medical_record.diagnoses" />
											<span className="text-xs font-mono bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300 px-1.5 py-0.5 rounded">
												{diagnosis_system === "cie10" ? "CIE-10" : "CIE-11"}
											</span>
											<span className="text-xs text-muted-foreground font-normal">
												(<Text uuid="form.optional" />)
											</span>
										</CardTitle>
									</div>
								</div>
							</CardHeader>
							<CardContent>
								<FormIcd11Select
									field={field}
									name="diagnoses"
									isOptional
									system={diagnosis_system}
								/>
							</CardContent>
						</Card>
					)}

					<Card className="border-l-4 border-l-violet-500">
						<CardHeader>
							<div className="flex items-center gap-3">
								<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-violet-500 text-white">
									<Pill className="h-4 w-4" />
								</div>
								<div>
									<CardTitle className="text-base">
										<Text uuid="form.create_medical_record.prescription.title" />
									</CardTitle>
									<p className="text-sm text-muted-foreground">
										<Text uuid="form.create_medical_record.prescription.description" />
									</p>
								</div>
							</div>
						</CardHeader>
						<CardContent>
							<Tabs
								value={prescriptionMode}
								onValueChange={(v) =>
									onPrescriptionModeChange(v as PrescriptionMode)
								}
							>
								<TabsList>
									<TabsTrigger value="text">
										<Text uuid="form.create_medical_record.prescription.tab.text" />
									</TabsTrigger>
									<TabsTrigger value="structured">
										<Text uuid="form.create_medical_record.prescription.tab.structured" />
									</TabsTrigger>
								</TabsList>

								<TabsContent value="text" className="space-y-4 pt-4">
									<FormTextArea
										field={field}
										name="prescription.content"
										label={
											<Text uuid="form.create_medical_record.prescription.content" />
										}
										isOptional
										rows={4}
									/>
									<FormTextArea
										field={field}
										name="prescription.indications"
										label={
											<Text uuid="form.create_medical_record.prescription.indications" />
										}
										isOptional
										rows={3}
									/>
								</TabsContent>

								<TabsContent value="structured" className="pt-4 space-y-3">
									<div className="flex justify-end">
										<Button
											type="button"
											variant="outline"
											size="sm"
											onClick={() =>
												append({
													medication: "",
													dose: "",
													frequency: "",
													duration: "",
													notes: "",
												})
											}
										>
											<Plus className="mr-2 h-4 w-4" />
											<Text uuid="form.create_medical_record.prescription.add_item" />
										</Button>
									</div>
									{fields.length === 0 && (
										<p className="text-sm text-muted-foreground text-center py-8 border border-dashed rounded-lg">
											<Text uuid="form.create_medical_record.prescription.items.empty" />
										</p>
									)}
									{fields.map((f, index) => (
										<div
											key={f.id}
											className="border rounded-lg p-4 space-y-3 bg-muted/20"
										>
											<div className="flex items-center justify-between mb-1">
												<span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
													Medicamento {index + 1}
												</span>
												<Button
													type="button"
													variant="ghost"
													size="sm"
													className="h-7 w-7 p-0 text-destructive hover:text-destructive"
													onClick={() => remove(index)}
												>
													<Trash className="h-3.5 w-3.5" />
												</Button>
											</div>
											<div className="grid md:grid-cols-2 grid-cols-1 gap-3">
												<FormInput
													field={field}
													name={`prescription.items.${index}.medication`}
													label={textGet(
														"form.create_medical_record.prescription.item.medication",
													)}
													placeholder="Amoxicilina 500mg"
												/>
												<FormInput
													field={field}
													name={`prescription.items.${index}.dose`}
													label={textGet(
														"form.create_medical_record.prescription.item.dose",
													)}
													placeholder="1 tableta"
												/>
												<FormInput
													field={field}
													name={`prescription.items.${index}.frequency`}
													label={textGet(
														"form.create_medical_record.prescription.item.frequency",
													)}
													placeholder="cada 8 horas"
												/>
												<FormInput
													field={field}
													name={`prescription.items.${index}.duration`}
													label={textGet(
														"form.create_medical_record.prescription.item.duration",
													)}
													placeholder="7 días"
												/>
											</div>
											<FormInput
												field={field}
												name={`prescription.items.${index}.notes`}
												label={textGet(
													"form.create_medical_record.prescription.item.notes",
												)}
												placeholder="Tomar con alimentos..."
												isOptional
											/>
										</div>
									))}
								</TabsContent>
							</Tabs>
						</CardContent>
					</Card>
				</TabsContent>
			</Tabs>

			{/* Sentinel — detected to know when footer is sticky */}
			<div ref={sentinelRef} className="h-px" />

			{/* Sticky bottom action bar */}
			<div
				className={`sticky z-10 bg-background/95 backdrop-blur-sm px-4 py-3 transition-all ${
					footerStuck
						? "bottom-3 -mx-1 border border-border rounded-2xl shadow-[0_4px_24px_rgba(0,0,0,0.10)] dark:shadow-[0_4px_24px_rgba(0,0,0,0.35)]"
						: "bottom-0 -mx-1"
				}`}
			>
				<div className="flex items-center justify-between gap-3">
					{/* Left: last record preview */}
					<div className="flex items-center gap-2 shrink-0">
						{lastRecord && (
							<Button
								type="button"
								variant="ghost"
								size="sm"
								className="text-muted-foreground"
								onClick={onPreviewLastRecord}
							>
								<Eye className="mr-1.5 h-4 w-4" />
								<span className="hidden sm:inline">
									<Text uuid="form.create_medical_record.preview_last" />
								</span>
							</Button>
						)}
					</div>

					{/* Center: tab step dots */}
					<div className="flex items-center gap-2">
						{TABS.map((tab) => (
							<button
								key={tab}
								type="button"
								onClick={() => goTo(tab)}
								className={`rounded-full transition-all ${
									activeTab === tab
										? "h-2.5 w-2.5 bg-primary"
										: tabDone[tab]
											? "h-2 w-2 bg-emerald-500"
											: "h-2 w-2 bg-muted-foreground/25"
								}`}
							/>
						))}
					</div>

					{/* Right: navigation + submit */}
					<div className="flex items-center gap-2 shrink-0">
						{!isFirst && (
							<Button
								type="button"
								variant="outline"
								size="sm"
								onClick={goPrev}
							>
								<ChevronLeft className="h-4 w-4" />
								<span className="hidden sm:inline ml-1">Anterior</span>
							</Button>
						)}
						{!isLast ? (
							<Button type="button" size="sm" onClick={goNext}>
								<span className="hidden sm:inline mr-1">Siguiente</span>
								<ChevronRight className="h-4 w-4" />
							</Button>
						) : (
							<Button type="submit" disabled={loading} size="sm">
								<Plus className="mr-1.5 h-4 w-4" />
								<Text uuid="form.create_medical_record.submit" />
								
							</Button>
						)}
					</div>
				</div>
			</div>
		</div>
	);
}

function LastRecordDialog({
	record,
	open,
	onClose,
	textGet,
}: {
	record: MedicalRecord;
	open: boolean;
	onClose: () => void;
	textGet: (key: string) => string;
}) {
	const date = new Date(record.date).toLocaleDateString();

	return (
		<Dialog open={open} onOpenChange={(v) => !v && onClose()}>
			<DialogContent className="sm:max-w-3xl max-h-[80vh] overflow-y-auto">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Eye className="h-4 w-4" />
						<Text uuid="form.create_medical_record.preview_last.title" />
						<span className="text-sm font-normal text-muted-foreground">
							— {date}
						</span>
					</DialogTitle>
				</DialogHeader>

				<div className="space-y-4 pt-2">
					<div>
						<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-1">
							<Text uuid="form.create_medical_record.motive" />
						</p>
						<p className="text-sm">{record.motive}</p>
					</div>

					{record.observation && (
						<div>
							<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-1">
								<Text uuid="form.create_medical_record.observation" />
							</p>
							<p className="text-sm">{record.observation}</p>
						</div>
					)}

					{record.soap_record && (
						<div className="grid gap-3 border rounded-md p-4">
							<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
								SOAP
							</p>
							{(["subjective", "objective", "assessment", "plan"] as const).map(
								(key) =>
									record.soap_record?.[key] ? (
										<div key={key}>
											<p className="text-xs font-medium capitalize text-muted-foreground mb-0.5">
												{textGet(`form.create_medical_record.soap.${key}`)}
											</p>
											<p className="text-sm whitespace-pre-wrap">
												{record.soap_record[key]}
											</p>
										</div>
									) : null,
							)}
						</div>
					)}

					{record.vital_signs && (
						<div className="border rounded-md p-4">
							<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">
								<Text uuid="form.create_medical_record.vital_signs.title" />
							</p>
							<div className="grid grid-cols-3 gap-2 text-sm">
								{record.vital_signs.weight && (
									<span>
										{textGet("form.create_medical_record.vital_signs.weight")}:{" "}
										<strong>{record.vital_signs.weight} kg</strong>
									</span>
								)}
								{record.vital_signs.height && (
									<span>
										{textGet("form.create_medical_record.vital_signs.height")}:{" "}
										<strong>{record.vital_signs.height} cm</strong>
									</span>
								)}
								{record.vital_signs.blood_pressure && (
									<span>
										{textGet(
											"form.create_medical_record.vital_signs.blood_pressure",
										)}
										: <strong>{record.vital_signs.blood_pressure}</strong>
									</span>
								)}
								{record.vital_signs.temperature && (
									<span>
										{textGet(
											"form.create_medical_record.vital_signs.temperature",
										)}
										: <strong>{record.vital_signs.temperature} °C</strong>
									</span>
								)}
								{record.vital_signs.heart_rate && (
									<span>
										{textGet(
											"form.create_medical_record.vital_signs.heart_rate",
										)}
										: <strong>{record.vital_signs.heart_rate} bpm</strong>
									</span>
								)}
								{record.vital_signs.o2_saturation && (
									<span>
										{textGet(
											"form.create_medical_record.vital_signs.o2_saturation",
										)}
										: <strong>{record.vital_signs.o2_saturation}%</strong>
									</span>
								)}
							</div>
						</div>
					)}

					{record.diagnoses && record.diagnoses.length > 0 && (
						<div>
							<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">
								<Text uuid="form.create_medical_record.diagnoses" />
							</p>
							<div className="flex flex-wrap gap-2">
								{record.diagnoses.map((d) => (
									<Badge key={d.code} variant="secondary">
										<span className="font-mono mr-1">{d.code}</span> {d.title}
									</Badge>
								))}
							</div>
						</div>
					)}

					{record.prescription &&
						(record.prescription.content ||
							record.prescription.indications) && (
							<div className="border rounded-md p-4">
								<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">
									<Text uuid="form.create_medical_record.prescription.title" />
								</p>
								{record.prescription.content && (
									<p className="text-sm whitespace-pre-wrap">
										{record.prescription.content}
									</p>
								)}
								{record.prescription.indications && (
									<p className="text-sm text-muted-foreground whitespace-pre-wrap mt-1">
										{record.prescription.indications}
									</p>
								)}
							</div>
						)}
				</div>
			</DialogContent>
		</Dialog>
	);
}

export default CreateMedicalRecordForm;