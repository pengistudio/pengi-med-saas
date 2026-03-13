import {
	Activity,
	AlertTriangle,
	Calendar,
	ClipboardList,
	FileText,
	Pill,
	Plus,
	Stethoscope,
	Trash,
} from "lucide-react";
import React from "react";
import { useFieldArray } from "react-hook-form";
import { useNavigate, useSearchParams } from "react-router";
import { z } from "zod";
import { createMedicalRecord } from "@/api/clinical-service";
import { Form } from "@/components/forms/form";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { FormCalendar } from "@/components/forms/form-calendar";
import { FormInput } from "@/components/forms/form-input";
import { FormTextArea } from "@/components/forms/form-textarea";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Tabs,
	TabsContent,
	TabsList,
	TabsTrigger,
} from "@/components/ui/tabs";
import { Text } from "@/components/ui/text";
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
		subjective: z.string({ error: "Campo requerido" }).min(1, "Campo requerido"),
		objective: z.string({ error: "Campo requerido" }).min(1, "Campo requerido"),
		assessment: z.string({ error: "Campo requerido" }).min(1, "Campo requerido"),
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
			o2_saturation: z.coerce.number().int().min(0).max(100).optional().nullable(),
		})
		.optional(),
});

const CreateMedicalRecordForm = () => {
	const [loading, setLoading] = React.useState(false);
	const [prescriptionMode, setPrescriptionMode] = React.useState<PrescriptionMode>("text");
	const { textGet } = useText();
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const patientId = searchParams.get("patient_id");
	const { patient } = usePatientStore();
	const allergies = parseAllergies(patient?.allergies);

	return (
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
				/>
			)}
		</Form>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!patientId) return;
		setLoading(true);

		let prescription: typeof values.prescription = undefined;

		if (prescriptionMode === "text") {
			const hasText = values.prescription?.content || values.prescription?.indications;
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
				// Map structured items to PDF fields:
				// content = medication names (one per line)
				// indications = instructions per medication (one per line)
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

		const res = await createMedicalRecord(payload);
		if (res.success) {
			navigate(`/clinical/medical-records/${patientId}`);
		}
		setLoading(false);
	}
};

function parseAllergies(allergies?: string): string[] {
	if (!allergies?.trim()) return [];
	return allergies.split(",").map((a) => a.trim()).filter(Boolean);
}

// biome-ignore lint/suspicious/noExplicitAny: UseFormReturn<any> is acceptable
function FormInner({
	field,
	loading,
	textGet,
	prescriptionMode,
	onPrescriptionModeChange,
	allergies,
}: {
	// biome-ignore lint/suspicious/noExplicitAny: <explanation>
	field: any;
	loading: boolean;
	textGet: (key: string) => string;
	prescriptionMode: PrescriptionMode;
	onPrescriptionModeChange: (mode: PrescriptionMode) => void;
	allergies: string[];
}) {
	const { fields, append, remove } = useFieldArray({
		name: "prescription.items",
		control: field.control,
	});

	return (
		<div className="space-y-4 max-w-4xl mx-auto">
			{/* Allergy Alert */}
			{allergies.length > 0 && (
				<Alert className="border-amber-300 bg-amber-50 dark:bg-amber-950/20 text-amber-900 dark:text-amber-200">
					<AlertTriangle className="text-amber-600!" />
					<AlertTitle>
						<Text uuid="form.create_medical_record.allergy_alert.title" />
					</AlertTitle>
					<AlertDescription className="space-y-2 text-amber-700 dark:text-amber-400">
						<p><Text uuid="form.create_medical_record.allergy_alert.description" /></p>
						<div className="flex flex-wrap gap-2">
							{allergies.map((a) => (
								<Badge
									key={a}
									variant="outline"
									className="bg-amber-100 text-amber-800 border-amber-400"
								>
									{a}
								</Badge>
							))}
						</div>
					</AlertDescription>
				</Alert>
			)}

			{/* General Info */}
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
					<div className="grid md:grid-cols-2 grid-cols-1 gap-2 md:gap-4">
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
						placeholder={textGet("form.create_medical_record.motive.placeholder")}
						description={textGet("form.create_medical_record.motive.description")}
					/>
					<FormTextArea
						field={field}
						name="observation"
						label={<Text uuid="form.create_medical_record.observation" />}
						placeholder={textGet("form.create_medical_record.observation.placeholder")}
						isOptional
					/>
				</CardContent>
			</Card>

			{/* Vital Signs */}
			<Card className="border-l-4 border-l-rose-500">
				<CardHeader>
					<div className="flex items-center gap-3">
						<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-rose-500 text-primary-foreground">
							<Activity className="h-4 w-4" />
						</div>
						<div>
							<CardTitle className="text-lg">
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
						<FormInput field={field} name="vital_signs.weight" type="number" step="0.1" label={textGet("form.create_medical_record.vital_signs.weight")} placeholder="kg" isOptional />
						<FormInput field={field} name="vital_signs.height" type="number" step="0.1" label={textGet("form.create_medical_record.vital_signs.height")} placeholder="cm" isOptional />
						<FormInput field={field} name="vital_signs.blood_pressure" label={textGet("form.create_medical_record.vital_signs.blood_pressure")} placeholder="120/80" isOptional />
						<FormInput field={field} name="vital_signs.temperature" type="number" step="0.1" label={textGet("form.create_medical_record.vital_signs.temperature")} placeholder="°C" isOptional />
						<FormInput field={field} name="vital_signs.heart_rate" type="number" label={textGet("form.create_medical_record.vital_signs.heart_rate")} placeholder="bpm" isOptional />
						<FormInput field={field} name="vital_signs.o2_saturation" type="number" label={textGet("form.create_medical_record.vital_signs.o2_saturation")} placeholder="%" isOptional />
					</div>
				</CardContent>
			</Card>

			{/* SOAP: Subjective */}
			<Card className="border-l-4 border-l-primary">
				<CardHeader>
					<div className="flex items-center gap-3">
						<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-primary-foreground">
							<FileText className="h-4 w-4" />
						</div>
						<div>
							<CardTitle className="text-lg">
								<Text uuid="form.create_medical_record.soap.subjective.title" />
							</CardTitle>
							<p className="text-sm text-muted-foreground">
								<Text uuid="form.create_medical_record.soap.subjective.description" />
							</p>
						</div>
					</div>
				</CardHeader>
				<CardContent>
					<FormTextArea field={field} name="soap_record.subjective" label={<Text uuid="form.create_medical_record.soap.subjective" />} />
				</CardContent>
			</Card>

			{/* SOAP: Objective */}
			<Card className="border-l-4 border-l-emerald-400">
				<CardHeader>
					<div className="flex items-center gap-3">
						<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-400 text-primary-foreground">
							<Stethoscope className="h-4 w-4" />
						</div>
						<div>
							<CardTitle className="text-lg">
								<Text uuid="form.create_medical_record.soap.objective.title" />
							</CardTitle>
							<p className="text-sm text-muted-foreground">
								<Text uuid="form.create_medical_record.soap.objective.description" />
							</p>
						</div>
					</div>
				</CardHeader>
				<CardContent>
					<FormTextArea field={field} name="soap_record.objective" label={<Text uuid="form.create_medical_record.soap.objective" />} />
				</CardContent>
			</Card>

			{/* SOAP: Assessment */}
			<Card className="border-l-4 border-l-orange-500">
				<CardHeader>
					<div className="flex items-center gap-3">
						<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-orange-500 text-primary-foreground">
							<ClipboardList className="h-4 w-4" />
						</div>
						<div>
							<CardTitle className="text-lg">
								<Text uuid="form.create_medical_record.soap.assessment.title" />
							</CardTitle>
							<p className="text-sm text-muted-foreground">
								<Text uuid="form.create_medical_record.soap.assessment.description" />
							</p>
						</div>
					</div>
				</CardHeader>
				<CardContent>
					<FormTextArea field={field} name="soap_record.assessment" label={<Text uuid="form.create_medical_record.soap.assessment" />} />
				</CardContent>
			</Card>

			{/* SOAP: Plan */}
			<Card className="border-l-4 border-l-teal-400">
				<CardHeader>
					<div className="flex items-center gap-3">
						<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-teal-400 text-primary-foreground">
							<Calendar className="h-4 w-4" />
						</div>
						<div>
							<CardTitle className="text-lg">
								<Text uuid="form.create_medical_record.soap.plan.title" />
							</CardTitle>
							<p className="text-sm text-muted-foreground">
								<Text uuid="form.create_medical_record.soap.plan.description" />
							</p>
						</div>
					</div>
				</CardHeader>
				<CardContent>
					<FormTextArea field={field} name="soap_record.plan" label={<Text uuid="form.create_medical_record.soap.plan" />} />
				</CardContent>
			</Card>

			{/* Prescription (optional) */}
			<Card className="border-l-4 border-l-violet-500">
				<CardHeader>
					<div className="flex items-center gap-3">
						<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-violet-500 text-primary-foreground">
							<Pill className="h-4 w-4" />
						</div>
						<div>
							<CardTitle className="text-lg">
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
						onValueChange={(v) => onPrescriptionModeChange(v as PrescriptionMode)}
					>
						<TabsList>
							<TabsTrigger value="text">
								<Text uuid="form.create_medical_record.prescription.tab.text" />
							</TabsTrigger>
							<TabsTrigger value="structured">
								<Text uuid="form.create_medical_record.prescription.tab.structured" />
							</TabsTrigger>
						</TabsList>

						{/* Free text tab */}
						<TabsContent value="text" className="space-y-4 pt-4">
							<FormTextArea
								field={field}
								name="prescription.content"
								label={<Text uuid="form.create_medical_record.prescription.content" />}
								isOptional
							/>
							<FormTextArea
								field={field}
								name="prescription.indications"
								label={<Text uuid="form.create_medical_record.prescription.indications" />}
								isOptional
							/>
						</TabsContent>

						{/* Structured tab */}
						<TabsContent value="structured" className="pt-4 space-y-3">
							<div className="flex justify-end">
								<Button
									type="button"
									variant="outline"
									size="sm"
									onClick={() =>
										append({ medication: "", dose: "", frequency: "", duration: "", notes: "" })
									}
								>
									<Plus className="mr-2 h-4 w-4" />
									<Text uuid="form.create_medical_record.prescription.add_item" />
								</Button>
							</div>
							{fields.length === 0 && (
								<p className="text-sm text-muted-foreground text-center py-6">
									<Text uuid="form.create_medical_record.prescription.items.empty" />
								</p>
							)}
							{fields.map((f, index) => (
								<div key={f.id} className="border rounded-md p-4 space-y-3">
									<div className="grid md:grid-cols-2 grid-cols-1 gap-3">
										<FormInput
											field={field}
											name={`prescription.items.${index}.medication`}
											label={textGet("form.create_medical_record.prescription.item.medication")}
											placeholder="Amoxicilina 500mg"
										/>
										<FormInput
											field={field}
											name={`prescription.items.${index}.dose`}
											label={textGet("form.create_medical_record.prescription.item.dose")}
											placeholder="1 tableta"
										/>
										<FormInput
											field={field}
											name={`prescription.items.${index}.frequency`}
											label={textGet("form.create_medical_record.prescription.item.frequency")}
											placeholder="cada 8 horas"
										/>
										<FormInput
											field={field}
											name={`prescription.items.${index}.duration`}
											label={textGet("form.create_medical_record.prescription.item.duration")}
											placeholder="7 días"
										/>
									</div>
									<div className="flex items-end gap-2">
										<div className="flex-1">
											<FormInput
												field={field}
												name={`prescription.items.${index}.notes`}
												label={textGet("form.create_medical_record.prescription.item.notes")}
												placeholder="..."
												isOptional
											/>
										</div>
										<Button
											type="button"
											variant="destructive"
											size="icon"
											onClick={() => remove(index)}
										>
											<Trash className="h-4 w-4" />
										</Button>
									</div>
								</div>
							))}
						</TabsContent>
					</Tabs>
				</CardContent>
			</Card>

			<div className="flex justify-end">
				<Button type="submit" disabled={loading}>
					<Plus className="mr-2 h-4 w-4" />
					<Text uuid="form.create_medical_record.submit" />
				</Button>
			</div>
		</div>
	);
}

export default CreateMedicalRecordForm;
