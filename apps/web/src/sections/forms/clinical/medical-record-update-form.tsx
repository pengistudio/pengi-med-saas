import {
	Calendar,
	Check,
	ClipboardList,
	FileText,
	Stethoscope,
} from "lucide-react";
import React from "react";
import { useNavigate, useParams } from "react-router";
import { z } from "zod";
import {
	getMedicalRecordById,
	type MedicalRecord,
	updateMedicalRecord,
} from "@/api/clinical-service";
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
});

const UpdateMedicalRecordForm = () => {
	const { id } = useParams<{ id: string }>();
	const [loading, setLoading] = React.useState(false);
	const [initialData, setInitialData] = React.useState<z.infer<
		typeof formSchema
	> | null>(null);
	const { errorToast } = useToast();
	const { textGet } = useText();
	const navigate = useNavigate();

	React.useEffect(() => {
		if (!id) return;

		getMedicalRecordById(Number(id)).then((res) => {
			if (!res.success) {
				errorToast(null, res.message);
				navigate(-1 as unknown as string);
				return;
			}
			const record = res.data as MedicalRecord;
			setInitialData({
				date: new Date(record.date),
				motive: record.motive,
				observation: record.observation || "",
				soap_record: {
					subjective: record.soap_record?.subjective || "",
					objective: record.soap_record?.objective || "",
					assessment: record.soap_record?.assessment || "",
					plan: record.soap_record?.plan || "",
				},
			});
		});
	}, [id, errorToast, navigate]);

	if (!initialData) {
		return (
			<div className="flex items-center justify-center h-64">
				<p className="text-muted-foreground">
					<Text uuid="form.update_medical_record.loading" />
				</p>
			</div>
		);
	}

	return (
		<Form schema={formSchema} defaultValues={initialData} onSubmit={onSubmit}>
			{(field) => (
				<div className="space-y-4 max-w-4xl mx-auto">
					{/* General Info */}
					<Card>
						<CardHeader>
							<CardTitle>
								<Text uuid="form.update_medical_record.title" />
							</CardTitle>
							<CardDescription>
								<Text uuid="form.update_medical_record.description" />
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-4">
							<div className="grid md:grid-cols-2 grid-cols-1 gap-2 md:gap-4">
								<FormCalendar
									field={field}
									name="date"
									label={textGet("form.update_medical_record.date")}
								/>
							</div>
							<FormTextArea
								field={field}
								name="motive"
								label={<Text uuid="form.update_medical_record.motive" />}
								placeholder={textGet(
									"form.update_medical_record.motive.placeholder",
								)}
								description={textGet(
									"form.update_medical_record.motive.description",
								)}
							/>
							<FormTextArea
								field={field}
								name="observation"
								label={<Text uuid="form.update_medical_record.observation" />}
								placeholder={textGet(
									"form.update_medical_record.observation.placeholder",
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

					<div className="flex justify-end">
						<Button type="submit" disabled={loading}>
							<Check className="mr-2 h-4 w-4" />
							<Text uuid="form.update_medical_record.submit" />
						</Button>
					</div>
				</div>
			)}
		</Form>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!id) return;
		setLoading(true);

		const payload = {
			date: values.date.toISOString(),
			motive: values.motive,
			observation: values.observation || "",
			soap_record: values.soap_record,
		};

		const res = await updateMedicalRecord(Number(id), payload);
		if (!res.success) {
			errorToast(null, res.message);
		} else {
			navigate(-1 as unknown as string);
		}
		setLoading(false);
	}
};

export default UpdateMedicalRecordForm;
