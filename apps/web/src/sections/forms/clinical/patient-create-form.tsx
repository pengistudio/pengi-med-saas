import { Plus } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import { z } from "zod";
import { createPatient } from "@/api/clinical-service";
import { Form } from "@/components/forms/form";
import { FormCalendar } from "@/components/forms/form-calendar";
import { FormInput } from "@/components/forms/form-input";
import { FormRadioGroup } from "@/components/forms/form-radio-group";
import { FormSelect } from "@/components/forms/form-select";
import { FormTagInput } from "@/components/forms/form-tag-input";
import { FormTextArea } from "@/components/forms/form-textarea";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Text } from "@/components/ui/text";
import useTenantSettings from "@/hooks/use-tenant-settings";
import { useText } from "@/hooks/use-text";

const STATIC_INSTITUTIONS = [
	{ label: "Solca", value: "Solca" },
	{ label: "Fundacen", value: "Fundacen" },
	{ label: "Santa Isabel", value: "Santa Isabel" },
	{ label: "Privado", value: "Privado" },
];

const formSchema = z.object({
	document: z
		.string()
		.min(10, "Debe tener 10 caracteres")
		.max(10, "Debe tener 10 caracteres"),
	phone: z.string().optional(),
	email: z.union([z.literal(""), z.email()]).optional(),
	first_name: z.string().min(1, "No debe estar vacío"),
	last_name: z.string().min(1, "No debe estar vacío"),
	birth_date: z.date().optional(),
	age: z.coerce.number().int().min(0).max(150).optional(),
	notes: z.string().optional(),
	insurance: z.string().optional(),
	medic: z.string().min(1, "No debe estar vacío"),
	gender: z.string().optional(),
	institution: z.string(),
	app: z.string().optional(),
	apf: z.string().optional(),
	apqx: z.string().optional(),
	diagnosis: z.string().optional(),
	allergies: z.string().optional(),
});

function ageToDate(age: number): Date {
	const now = new Date();
	return new Date(now.getFullYear() - age, now.getMonth(), now.getDate());
}

const CreatePatientForm = () => {
	const [loading, setLoading] = React.useState(false);
	const navigate = useNavigate();
	const { settings } = useTenantSettings();
	const { textGet } = useText();
	const useAgeInput = settings.clinical.patient_age_input;

	return (
		<Form
			schema={formSchema}
			onSubmit={onSubmit}
			defaultValues={{
				medic: "",
			}}
		>
			{(field) => (
				<Card className="max-w-4xl mx-auto">
					<CardHeader>
						<CardTitle>
							<Text uuid="form.create_patient.title" />
						</CardTitle>
						<CardDescription>
							<Text uuid="form.create_patient.description" />
						</CardDescription>
					</CardHeader>
					<CardContent className="space-y-4">
						<div className="grid md:grid-cols-3 grid-cols-1 gap-2 md:gap-4">
							<FormInput
								field={field}
								name="document"
								placeholder={textGet("form.edit_patient.document.placeholder")}
								label={textGet("form.edit_patient.document")}
							/>
							<FormInput
								field={field}
								name="first_name"
								placeholder={textGet(
									"form.edit_patient.first_name.placeholder",
								)}
								label={textGet("form.edit_patient.first_name")}
							/>
							<FormInput
								field={field}
								name="last_name"
								placeholder={textGet("form.edit_patient.last_name.placeholder")}
								label={textGet("form.edit_patient.last_name")}
							/>
						</div>
						<div className="grid md:grid-cols-2 grid-cols-1 gap-2 md:gap-4">
							<FormInput
								field={field}
								name="phone"
								placeholder={textGet("form.edit_patient.phone.placeholder")}
								label={textGet("form.edit_patient.phone")}
								isOptional
							/>
							<FormInput
								field={field}
								name="email"
								type="email"
								placeholder={textGet("form.edit_patient.email.placeholder")}
								label={textGet("form.edit_patient.email")}
								isOptional
							/>

							{useAgeInput ? (
								<FormInput
									field={field}
									name="age"
									type="number"
									placeholder={textGet("form.patient.age.placeholder")}
									label={textGet("form.patient.age")}
									isOptional
								/>
							) : (
								<FormCalendar
									field={field}
									name="birth_date"
									label={textGet("form.edit_patient.birth_date")}
									isOptional
									showMonthYearDropdowns
								/>
							)}

							<FormRadioGroup
								name="gender"
								label={textGet("form.edit_patient.gender")}
								field={field}
								isRow
								options={[
									{ label: "Masculino", value: "M" },
									{ label: "Femenino", value: "F" },
								]}
							/>
							<FormSelect
								name="institution"
								label={textGet("form.edit_patient.institution")}
								placeholder={textGet(
									"form.edit_patient.institution.placeholder",
								)}
								field={field}
								options={STATIC_INSTITUTIONS}
							/>
							<FormInput
								field={field}
								name="medic"
								placeholder={textGet("form.edit_patient.medic.placeholder")}
								label={textGet("form.edit_patient.medic")}
							/>
							<FormInput
								field={field}
								name="insurance"
								placeholder={textGet("form.edit_patient.insurance.placeholder")}
								label={textGet("form.edit_patient.insurance")}
								isOptional
							/>
						</div>

						<FormTextArea
							field={field}
							name="diagnosis"
							placeholder={textGet("form.edit_patient.diagnosis.placeholder")}
							label={<Text uuid="form.edit_patient.diagnosis" />}
							isOptional
						/>

						<FormTextArea
							field={field}
							name="app"
							placeholder="APP..."
							label={<Text uuid="form.edit_patient.app" />}
							isOptional
						/>

						<FormTextArea
							field={field}
							name="apf"
							placeholder="APF..."
							label={<Text uuid="form.edit_patient.apf" />}
							isOptional
						/>

						<FormTextArea
							field={field}
							name="apqx"
							placeholder="APQX..."
							label={<Text uuid="form.edit_patient.apqx" />}
							isOptional
						/>

						<FormTagInput
							field={field}
							name="allergies"
							placeholder={textGet("form.edit_patient.allergies.placeholder")}
							label={textGet("form.edit_patient.allergies")}
							isOptional
						/>

						<FormTextArea
							field={field}
							name="notes"
							placeholder={textGet("form.edit_patient.notes.placeholder")}
							label={<Text uuid="form.edit_patient.notes" />}
							isOptional
						/>
					</CardContent>
					<CardFooter>
						<Button type="submit" disabled={loading}>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="form.create_patient.submit" />
						</Button>
					</CardFooter>
				</Card>
			)}
		</Form>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoading(true);

		const birth_date =
			useAgeInput && values.age !== undefined
				? ageToDate(values.age)
				: values.birth_date;

		const { age: _age, ...rest } = values;
		const payload = { ...rest, birth_date };

		try {
			const res = await createPatient(payload);
			if (res.success) {
				navigate("/clinical");
			}
		} finally {
			setLoading(false);
		}
	}
};

export default CreatePatientForm;
