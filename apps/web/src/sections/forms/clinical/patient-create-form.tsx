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
import useToast from "@/hooks/use-toast";

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
	first_name: z.string().min(1, "No debe estar vacío"),
	last_name: z.string().min(1, "No debe estar vacío"),
	birth_date: z.date().optional(),
	next_appointment: z
		.object({
			date: z.date().optional(),
			startTime: z.string().optional(),
			endTime: z.string().optional(),
		})
		.optional(),
	notes: z.string().optional(),
	insurance: z.string().optional(),
	medic: z.string().min(1, "No debe estar vacío"),
	gender: z.string().optional(),
	institution: z.string(),
	app: z.string().optional(),
	apf: z.string().optional(),
	apqx: z.string().optional(),
	diagnosis: z.string().optional(),
});

const CreatePatientForm = () => {
	const [loading, setLoading] = React.useState(false);
	const { errorToast } = useToast();

	const navigate = useNavigate();

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
						<CardTitle>Crear Paciente</CardTitle>
						<CardDescription>
							Llena los campos para crear un paciente
						</CardDescription>
					</CardHeader>
					<CardContent className="space-y-4">
						<div className="grid md:grid-cols-3 grid-cols-1 gap-2 md:gap-4">
							<FormInput
								field={field}
								name="document"
								placeholder="Cédula..."
								label="Cédula"
							/>
							<FormInput
								field={field}
								name="first_name"
								placeholder="Nombre..."
								label="Nombre"
							/>
							<FormInput
								field={field}
								name="last_name"
								placeholder="Apellido..."
								label="Apellido"
							/>
						</div>
						<div className="grid md:grid-cols-2 grid-cols-1 gap-2 md:gap-4">
							<FormInput
								field={field}
								name="phone"
								placeholder="Teléfono..."
								label="Teléfono"
								isOptional
							/>

							<FormCalendar
								field={field}
								name="birth_date"
								label="Fecha de Nacimiento"
								isOptional
							/>

							<FormCalendar
								field={field}
								name="next_appointment"
								label="Próxima Cita"
								enableTime={true}
								isOptional
							/>

							<FormRadioGroup
								name="gender"
								label="Género"
								field={field}
								isRow
								options={[
									{ label: "Masculino", value: "M" },
									{ label: "Femenino", value: "F" },
								]}
							/>
							<FormSelect
								name="institution"
								label="Institución"
								placeholder="Seleccionar Institución..."
								field={field}
								options={STATIC_INSTITUTIONS}
							/>
							<FormInput
								field={field}
								name="medic"
								placeholder="Dr. o Dra..."
								label="Médico Tratante"
							/>
							<FormInput
								field={field}
								name="insurance"
								placeholder="Seguro Médico..."
								label="Seguro Médico"
								isOptional
							/>
						</div>

						<FormTextArea
							field={field}
							name="diagnosis"
							placeholder="Diagnóstico..."
							label="Diagnóstico"
							isOptional
						/>

						<FormTextArea
							field={field}
							name="app"
							placeholder="APP..."
							label="Antecedentes Personales Patológicos"
							isOptional
						/>

						<FormTextArea
							field={field}
							name="apf"
							placeholder="APF (Antecedentes Patológicos Familiares)..."
							label="Antecedentes Patológicos Familiares"
							isOptional
						/>

						<FormTextArea
							field={field}
							name="apqx"
							placeholder="APQX (Antecedentes Patológicos Quirúrgicos)..."
							label="Antecedentes Patológicos Quirúrgicos"
							isOptional
						/>

						<FormTextArea
							field={field}
							name="notes"
							label="Notas"
							placeholder="Notas..."
							isOptional
						/>
					</CardContent>
					<CardFooter>
						<Button type="submit" disabled={loading}>
							<Plus className="mr-2 h-4 w-4" />
							Crear Paciente
						</Button>
					</CardFooter>
				</Card>
			)}
		</Form>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoading(true);

		const { next_appointment, ...restValues } = values;
		const payload = {
			...restValues,
			next_appointment_date: next_appointment?.date,
			next_appointment_start_time: next_appointment?.startTime,
			next_appointment_end_time: next_appointment?.endTime,
		};

		const res = await createPatient(payload);
		if (!res.success) {
			errorToast(null, res.message);
		} else {
			navigate("/clinical");
		}
		setLoading(false);
	}
};

export default CreatePatientForm;
