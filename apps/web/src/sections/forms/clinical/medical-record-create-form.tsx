import {
	Calendar,
	ClipboardList,
	FileText,
	Pill,
	Plus,
	Stethoscope,
} from "lucide-react";
import React from "react";
import { useNavigate, useSearchParams } from "react-router";
import { z } from "zod";
import { createMedicalRecord } from "@/api/clinical-service";
import { Form } from "@/components/forms/form";
import { FormCalendar } from "@/components/forms/form-calendar";
import { FormTextArea } from "@/components/forms/form-textarea";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";
import useToast from "@/hooks/use-toast";

const formSchema = z.object({
	date: z.date({ error: "Campo requerido" }),
	next_appointment: z
		.object({
			date: z.date().optional(),
			startTime: z.string().optional(),
			endTime: z.string().optional(),
		})
		.optional(),
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
		})
		.optional(),
});

const CreateMedicalRecordForm = () => {
	const [loading, setLoading] = React.useState(false);
	const { errorToast } = useToast();
	const { textGet } = useText();
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const patientId = searchParams.get("patient_id");

	return (
		<Form
			schema={formSchema}
			onSubmit={onSubmit}
			defaultValues={{
				date: new Date(),
			}}
		>
			{(field) => (
				<div className="space-y-4 max-w-4xl mx-auto">
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
								<FormCalendar
									field={field}
									name="next_appointment"
									label={textGet("form.create_medical_record.next_appointment")}
									isOptional
									enableTime
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
							/>
							<FormTextArea
								field={field}
								name="observation"
								label={<Text uuid="form.create_medical_record.observation" />}
								placeholder={textGet(
									"form.create_medical_record.observation.placeholder",
								)}
								isOptional
							/>
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
							<FormTextArea
								field={field}
								name="soap_record.subjective"
								label={
									<Text uuid="form.create_medical_record.soap.subjective" />
								}
							/>
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
							<FormTextArea
								field={field}
								name="soap_record.objective"
								label={
									<Text uuid="form.create_medical_record.soap.objective" />
								}
							/>
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
							<FormTextArea
								field={field}
								name="soap_record.assessment"
								label={
									<Text uuid="form.create_medical_record.soap.assessment" />
								}
							/>
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
							<FormTextArea
								field={field}
								name="soap_record.plan"
								label={<Text uuid="form.create_medical_record.soap.plan" />}
							/>
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
						<CardContent className="space-y-4">
							<FormTextArea
								field={field}
								name="prescription.content"
								label={
									<Text uuid="form.create_medical_record.prescription.content" />
								}
								isOptional
							/>
							<FormTextArea
								field={field}
								name="prescription.indications"
								label={
									<Text uuid="form.create_medical_record.prescription.indications" />
								}
								isOptional
							/>
						</CardContent>
					</Card>

					<div className="flex justify-end">
						<Button type="submit" disabled={loading}>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="form.create_medical_record.submit" />
						</Button>
					</div>
				</div>
			)}
		</Form>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!patientId) return;
		setLoading(true);

		const { next_appointment, ...restValues } = values;

		const payload = {
			patient_id: Number(patientId),
			date: restValues.date.toISOString(),
			motive: restValues.motive,
			observation: restValues.observation || "",
			next_appointment_date: next_appointment?.date?.toISOString(),
			next_appointment_start_time: next_appointment?.startTime,
			next_appointment_end_time: next_appointment?.endTime,
			soap_record: restValues.soap_record,
			prescription:
				restValues.prescription?.content || restValues.prescription?.indications
					? {
							content: restValues.prescription?.content || "",
							indications: restValues.prescription?.indications || "",
						}
					: undefined,
		};

		const res = await createMedicalRecord(payload);
		if (!res.success) {
			errorToast(null, res.message);
		} else {
			navigate(`/clinical/medical-records/${patientId}`);
		}
		setLoading(false);
	}
};

export default CreateMedicalRecordForm;
