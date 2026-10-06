import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	Form,
	FormCheckbox,
	FormInput,
	FormRadioGroup,
	FormSelect,
	FormTextArea,
	Text,
} from "@pengi/ui";
import { Save } from "lucide-react";
import React from "react";
import { useNavigate, useParams } from "react-router";
import { z } from "zod";
import {
	getPatientById,
	type Patient,
	updatePatient,
} from "@/api/clinical-service";
import { FormCalendar } from "@/components/forms/form-calendar";
import useTenantSettings from "@/hooks/use-tenant-settings";
import { ageInYears, birthDateFromAge } from "@/lib/patient-age";
import { selectSetPatient, usePatientStore } from "@/store/patient-store";

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
	whatsapp_opt_in: z.boolean().optional(),
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
});

const EditPatientForm = () => {
	const { id } = useParams<{ id: string }>();
	const [loading, setLoading] = React.useState(false);
	const [loadingData, setLoadingData] = React.useState(true);
	const [patient, setPatientState] = React.useState<Patient | null>(null);
	const { textGet } = useText();
	const { settings } = useTenantSettings();
	const setPatient = usePatientStore(selectSetPatient);
	const navigate = useNavigate();
	const useAgeInput = settings.clinical.patient_age_input;
	// With age input on, a patient with a real birth date keeps the date picker
	// so the exact date isn't replaced by an estimate.
	const [exactDate, setExactDate] = React.useState(false);

	React.useEffect(() => {
		if (!id) return;

		getPatientById(Number(id))
			.then((res) => {
				if (!res.success) {
					navigate("/clinical");
					return;
				}
				if (res.data) {
					const loaded = res.data as Patient;
					setPatientState(loaded);
					setExactDate(
						!loaded.birth_date_estimated &&
							ageInYears(loaded.birth_date) !== null,
					);
				}
			})
			.finally(() => setLoadingData(false));
	}, [id, navigate]);

	if (loadingData || !patient) {
		return (
			<Card className="max-w-4xl mx-auto">
				<CardHeader>
					<CardTitle>
						<Text uuid="form.edit_patient.loading" />
					</CardTitle>
				</CardHeader>
			</Card>
		);
	}

	const existingAge = ageInYears(patient.birth_date) ?? undefined;
	const existingBirthDate =
		existingAge !== undefined ? new Date(patient.birth_date) : undefined;

	const defaultValues = {
		document: patient.document,
		phone: patient.phone || "",
		whatsapp_opt_in: patient.whatsapp_opt_in ?? false,
		email: patient.email || "",
		first_name: patient.first_name,
		last_name: patient.last_name,
		birth_date: existingBirthDate,
		age: existingAge,
		notes: patient.notes || "",
		insurance: patient.insurance || "",
		medic: patient.medic,
		gender: patient.gender || "",
		institution: patient.institution || "",
	};

	return (
		<Form schema={formSchema} onSubmit={onSubmit} defaultValues={defaultValues}>
			{(field) => (
				<Card className="max-w-4xl mx-auto">
					<CardHeader>
						<CardTitle>
							<Text uuid="form.edit_patient.title" />
						</CardTitle>
						<CardDescription>
							{textGet("form.edit_patient.description")} {patient.full_name}
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

							<div className="md:col-span-2">
								<FormCheckbox
									field={field}
									name="whatsapp_opt_in"
									label={textGet("form.patient.whatsapp_opt_in")}
									description={textGet(
										"form.patient.whatsapp_opt_in.description",
									)}
								/>
							</div>

							{useAgeInput && !exactDate ? (
								<div className="space-y-1.5">
									<FormInput
										field={field}
										name="age"
										type="number"
										placeholder={textGet("form.patient.age.placeholder")}
										label={textGet("form.patient.age")}
										isOptional
									/>
									<button
										type="button"
										onClick={() => setExactDate(true)}
										className="text-xs text-primary underline-offset-4 hover:underline"
									>
										{textGet("form.patient.enter_exact_birth_date")}
									</button>
								</div>
							) : (
								<div className="space-y-1.5">
									<FormCalendar
										field={field}
										name="birth_date"
										label={textGet("form.edit_patient.birth_date")}
										isOptional
										showMonthYearDropdowns
									/>
									{useAgeInput && (
										<button
											type="button"
											onClick={() => setExactDate(false)}
											className="text-xs text-primary underline-offset-4 hover:underline"
										>
											{textGet("form.patient.enter_age_only")}
										</button>
									)}
									{patient.birth_date_estimated && (
										<p className="text-xs text-muted-foreground">
											{textGet("form.patient.birth_date_estimated_hint")}
										</p>
									)}
								</div>
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
							name="notes"
							placeholder={textGet("form.edit_patient.notes.placeholder")}
							label={<Text uuid="form.edit_patient.notes" />}
							isOptional
						/>
					</CardContent>
					<CardFooter>
						<Button type="submit" disabled={loading}>
							<Save className="mr-2 h-4 w-4" />
							<Text uuid="form.edit_patient.submit" />
						</Button>
					</CardFooter>
				</Card>
			)}
		</Form>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		if (!id) return;
		setLoading(true);

		// Send the birth date only when it changed, so saving other fields never
		// moves it or turns an exact date into an estimate.
		const byAge = useAgeInput && !exactDate;
		let birthDateChange = {};
		if (byAge) {
			if (values.age !== undefined && values.age !== existingAge) {
				birthDateChange = {
					birth_date: birthDateFromAge(values.age),
					birth_date_estimated: true,
				};
			}
		} else if (
			values.birth_date &&
			values.birth_date.getTime() !== existingBirthDate?.getTime()
		) {
			birthDateChange = {
				birth_date: values.birth_date,
				birth_date_estimated: false,
			};
		}

		const { age: _age, birth_date: _birthDate, ...rest } = values;
		const payload = { ...rest, ...birthDateChange };

		try {
			const res = await updatePatient(Number(id), payload);
			if (res.success) {
				if (res.data) {
					setPatient(res.data as Patient);
				}
				navigate(-1 as unknown as string);
			}
		} finally {
			setLoading(false);
		}
	}
};

export default EditPatientForm;
